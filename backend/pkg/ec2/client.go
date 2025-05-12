package ec2

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

type EC2Client struct {
	client *ec2.Client
	ctx    context.Context
}

func NewEC2Client() (*EC2Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	// Add this debug line
	fmt.Printf("Using AWS Region: %s\n", cfg.Region)

	return &EC2Client{
		client: ec2.NewFromConfig(cfg),
		ctx:    context.Background(),
	}, nil
}

func (c *EC2Client) CreateInstance(name, imageID, instanceType, subnetID, securityGroupID string) (string, error) {
	// Use the provided AMI ID
	fmt.Printf("Using AMI: %s (Amazon Linux 2)\n", imageID)

	// Create or get key pair
	keyName := "docker-sandbox-key"
	keyPath := "docker-sandbox-key.pem"

	// Check if key pair exists
	describeKeyInput := &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	}
	_, err := c.client.DescribeKeyPairs(c.ctx, describeKeyInput)
	if err != nil {
		// Create new key pair
		createKeyInput := &ec2.CreateKeyPairInput{
			KeyName: aws.String(keyName),
		}
		keyPair, err := c.client.CreateKeyPair(c.ctx, createKeyInput)
		if err != nil {
			return "", fmt.Errorf("failed to create key pair: %v", err)
		}

		// Save private key to file
		if err := os.WriteFile(keyPath, []byte(*keyPair.KeyMaterial), 0600); err != nil {
			return "", fmt.Errorf("failed to save private key: %v", err)
		}
		fmt.Printf("Created new key pair and saved private key to %s\n", keyPath)
	} else {
		fmt.Printf("Using existing key pair %s\n", keyName)
		// Check if we have the private key file
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			fmt.Printf("Warning: Private key file %s not found. You may need to recreate the key pair.\n", keyPath)
		}
	}

	// Check if AMI supports console output
	describeImageInput := &ec2.DescribeImagesInput{
		ImageIds: []string{imageID},
	}
	describeImageResult, err := c.client.DescribeImages(c.ctx, describeImageInput)
	if err != nil {
		fmt.Printf("Warning: Could not verify AMI console output support: %v\n", err)
	} else if len(describeImageResult.Images) > 0 {
		image := describeImageResult.Images[0]
		fmt.Printf("AMI Details:\n")
		fmt.Printf("  Name: %s\n", *image.Name)
		fmt.Printf("  Description: %s\n", *image.Description)
		fmt.Printf("  Platform: %s\n", image.Platform)
		fmt.Printf("  Architecture: %s\n", image.Architecture)
	}

	// Create security group first
	sgInput := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("DockerSandbox"),
		Description: aws.String("Security group for Docker sandbox"),
	}

	sgResult, err := c.client.CreateSecurityGroup(c.ctx, sgInput)
	if err != nil {
		// If security group already exists, try to find it
		describeSGInput := &ec2.DescribeSecurityGroupsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("group-name"),
					Values: []string{"DockerSandbox"},
				},
			},
		}
		describeSGResult, err := c.client.DescribeSecurityGroups(c.ctx, describeSGInput)
		if err != nil || len(describeSGResult.SecurityGroups) == 0 {
			return "", fmt.Errorf("failed to create or find security group: %v", err)
		}
		securityGroupID = *describeSGResult.SecurityGroups[0].GroupId
	} else {
		securityGroupID = *sgResult.GroupId
	}

	// Add inbound rule for Docker daemon
	_, err = c.client.AuthorizeSecurityGroupIngress(c.ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []types.IpPermission{
			{
				FromPort:   aws.Int32(2375),
				ToPort:     aws.Int32(2375),
				IpProtocol: aws.String("tcp"),
				IpRanges: []types.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
				},
			},
			// Add SSH access
			{
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpProtocol: aws.String("tcp"),
				IpRanges: []types.IpRange{
					{
						CidrIp: aws.String("0.0.0.0/0"),
					},
				},
			},
		},
	})
	if err != nil {
		// Check if the error is about duplicate rules
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidPermission.Duplicate" {
			// Ignore duplicate rule error
			fmt.Println("Security group rule already exists, continuing...")
		} else {
			return "", fmt.Errorf("failed to authorize security group ingress: %v", err)
		}
	}

	userData := `#!/bin/bash
echo "Starting user data script" > /var/log/user-data-start.log

# Wait for yum lock to be released
while fuser /var/run/yum.pid >/dev/null 2>&1; do
    echo "Waiting for yum lock..." >> /var/log/user-data-start.log
    sleep 3
done

# Update system
yum update -y >> /var/log/user-data-start.log 2>&1

# Install Docker
amazon-linux-extras install docker -y >> /var/log/user-data-start.log 2>&1
yum install -y docker >> /var/log/user-data-start.log 2>&1

# Start and enable Docker
systemctl start docker >> /var/log/user-data-start.log 2>&1
systemctl enable docker >> /var/log/user-data-start.log 2>&1

# Add ec2-user to docker group
usermod -aG docker ec2-user >> /var/log/user-data-start.log 2>&1

# Configure Docker daemon to listen on TCP port 2375
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << EOF
{
    "hosts": ["tcp://0.0.0.0:2375", "unix:///var/run/docker.sock"],
    "debug": true
}
EOF

# Override the default systemd service to remove -H fd://
mkdir -p /etc/systemd/system/docker.service.d
cat > /etc/systemd/system/docker.service.d/override.conf << EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd
EOF

# Reload systemd and restart Docker
systemctl daemon-reload
systemctl restart docker >> /var/log/user-data-start.log 2>&1

echo "User data script completed" >> /var/log/user-data-start.log
`

	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(imageID),
		InstanceType:     types.InstanceTypeT3Micro,
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		UserData:         aws.String(encodedUserData),
		SecurityGroupIds: []string{securityGroupID},
		KeyName:          aws.String(keyName),
		SubnetId:         aws.String(subnetID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("Docker-Sandbox"),
					},
				},
			},
		},
	}

	result, err := c.client.RunInstances(c.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create instance: %v", err)
	}

	if len(result.Instances) == 0 {
		return "", fmt.Errorf("no instance created")
	}

	instanceID := *result.Instances[0].InstanceId
	fmt.Printf("Created instance %s\n", instanceID)

	// Wait for instance to be ready and logs to be available
	if err := c.WaitForInstanceReady(instanceID); err != nil {
		// Get instance state before terminating
		describeResult, err := c.DescribeInstance(instanceID)
		if err == nil && len(describeResult.Reservations) > 0 && len(describeResult.Reservations[0].Instances) > 0 {
			instance := describeResult.Reservations[0].Instances[0]
			fmt.Printf("Instance state: %s\n", instance.State.Name)
			fmt.Printf("Instance type: %s\n", instance.InstanceType)
			fmt.Printf("Platform: %s\n", instance.Platform)
			if instance.PublicIpAddress != nil {
				fmt.Printf("Public IP: %s\n", *instance.PublicIpAddress)
			}
		}
		return instanceID, fmt.Errorf("instance failed to initialize: %v", err)
	}

	// Get and print the system logs
	logs, err := c.GetInstanceLogs(instanceID)
	if err != nil {
		fmt.Printf("Warning: Failed to get system logs: %v\n", err)
	} else {
		fmt.Println("System logs:")
		fmt.Println(logs)
	}

	// After instance is created and we have the public IP, print SSH instructions
	if len(result.Instances) > 0 && result.Instances[0].PublicIpAddress != nil {
		publicIP := *result.Instances[0].PublicIpAddress
		fmt.Printf("\nTo SSH into the instance:\n")
		fmt.Printf("1. Make sure the private key file %s has correct permissions:\n", keyPath)
		fmt.Printf("   chmod 400 %s\n", keyPath)
		fmt.Printf("2. Connect using:\n")
		fmt.Printf("   ssh -i %s ec2-user@%s\n", keyPath, publicIP)
	}

	return instanceID, nil
}

func (c *EC2Client) DescribeInstance(instanceID string) (*ec2.DescribeInstancesOutput, error) {
	return c.client.DescribeInstances(c.ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
}

func (ec *EC2Client) TerminateInstance(instanceID string) error {
	_, err := ec.client.TerminateInstances(ec.ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

func (c *EC2Client) WaitForInstanceReady(instanceID string) error {
	fmt.Printf("Waiting for instance %s to be ready...\n", instanceID)

	// First wait for instance to be running
	waiter := ec2.NewInstanceRunningWaiter(c.client)
	if err := waiter.Wait(c.ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 10*time.Minute); err != nil {
		return fmt.Errorf("instance failed to start: %v", err)
	}

	// Get instance details to verify it's running
	describeResult, err := c.DescribeInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to describe instance: %v", err)
	}

	if len(describeResult.Reservations) == 0 || len(describeResult.Reservations[0].Instances) == 0 {
		return fmt.Errorf("no instance found")
	}

	instance := describeResult.Reservations[0].Instances[0]
	fmt.Printf("Instance is running:\n")
	fmt.Printf("  State: %s\n", instance.State.Name)
	fmt.Printf("  Type: %s\n", instance.InstanceType)
	fmt.Printf("  Platform: %s\n", instance.Platform)
	if instance.PublicIpAddress != nil {
		fmt.Printf("  Public IP: %s\n", *instance.PublicIpAddress)
	}

	// Wait for instance status checks to pass
	fmt.Println("Waiting for instance status checks to pass...")
	statusWaiter := ec2.NewInstanceStatusOkWaiter(c.client)
	if err := statusWaiter.Wait(c.ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceID},
	}, 10*time.Minute); err != nil {
		return fmt.Errorf("instance status checks failed: %v", err)
	}
	fmt.Println("Instance status checks passed")

	// Give the user data script a moment to start
	fmt.Println("Waiting for user data script to initialize...")
	time.Sleep(30 * time.Second)

	return nil
}

func (c *EC2Client) GetInstanceLogs(instanceID string) (string, error) {
	output, err := c.client.GetConsoleOutput(c.ctx, &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get console output: %v", err)
	}

	if output.Output == nil {
		return "", fmt.Errorf("no console output available")
	}

	return *output.Output, nil
}
