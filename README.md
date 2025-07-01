# K0 - Rethinking Technical Interviews

> *"It just exposed how broken technical interviews are. Repetitive, predictable, and not really measuring what matters."*

K0 is revolutionizing technical interviews by moving beyond whiteboard coding and LeetCode grinding. Instead of artificial coding challenges, we enable recruiters and interviewers to explore candidates' **real projects** in live, shared environments.

## 🎯 The Problem We're Solving

Traditional technical interviews are fundamentally broken:
- **Whiteboard coding** doesn't reflect real development work
- **LeetCode challenges** test memorization over engineering skills  
- **Artificial environments** don't show how candidates actually build
- **No accountability** for claimed project experience

## 💡 Our Solution

K0 transforms technical evaluation through:

### Real Project Exploration
Simply paste a GitHub URL and instantly explore the candidate's actual codebase alongside a live deployment - all in one shared environment.

### Live Multiplayer Environments  
Multiple participants can simultaneously navigate, discuss, and interact with the deployed application in real-time.

### Concurrent Breakout Rooms
Support for multiple interview sessions running simultaneously, each with their own isolated environment.

### Dockerized Deployments
Automatic containerization and deployment of GitHub repositories, providing consistent and secure execution environments.

## 🚀 Key Features

- **🔗 One-Click GitHub Integration** - Paste any GitHub URL to instantly clone and deploy
- **🐳 Automatic Docker Deployment** - Seamless containerization of any project  
- **👥 Real-time Collaboration** - Multiple users can interact simultaneously
- **💬 Live Terminal Sharing** - Share terminal output in real-time via WebSockets
- **🏢 Room Management** - Isolated environments for different interview sessions
- **☁️ Cloud Infrastructure** - Scalable deployment on AWS infrastructure
- **📱 Modern UI** - Clean, responsive interface built with Next.js

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend       │    │   Infrastructure│
│   (Next.js)     │◄──►│   (Go/Fiber)    │◄──►│   (AWS/Docker)  │
│                 │    │                 │    │                 │
│ • React/TS      │    │ • WebSocket API │    │ • S3 Storage    │
│ • WebSocket     │    │ • Container Mgmt│    │ • EC2 Instances │
│ • Tailwind CSS  │    │ • GitHub Client │    │ • Docker Engine │
│ • Supabase      │    │ • Supabase      │    │ • Load Balancer │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🛠️ Tech Stack

### Frontend
- **Next.js 14** - React framework with App Router
- **TypeScript** - Type-safe development
- **Tailwind CSS** - Utility-first styling
- **WebSocket API** - Real-time communication
- **Supabase Client** - Database integration

### Backend  
- **Go** - High-performance backend language
- **Fiber** - Express-inspired web framework
- **WebSocket** - Real-time bidirectional communication
- **Docker SDK** - Container management
- **AWS SDK** - Cloud service integration

### Infrastructure
- **Docker** - Containerization platform
- **Amazon S3** - Object storage
- **Amazon EC2** - Compute instances  
- **Supabase** - PostgreSQL database with real-time features
- **GitHub API** - Repository integration

## 🚦 Getting Started

### Prerequisites

- **Go 1.21+**
- **Node.js 18+**
- **Docker Desktop**
- **AWS Account** (for cloud features)
- **Supabase Account**

### Backend Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd K0/backend
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run the server**
   ```bash
   go run cmd/server/main.go
   ```

The backend will start on `http://localhost:3009`

### Frontend Setup

1. **Navigate to frontend directory**
   ```bash
   cd client/k0-frontend
   ```

2. **Install dependencies**
   ```bash
   npm install
   ```

3. **Configure environment**
   ```bash
   cp .env.local.example .env.local
   # Edit .env.local with your configuration
   ```

4. **Start development server**
   ```bash
   npm run dev
   ```

The frontend will start on `http://localhost:3000`

## 📖 How It Works

### For Interviewers

1. **Create a Room** - Start a new interview session
2. **Share GitHub URL** - Paste the candidate's repository link
3. **Explore Together** - Navigate the codebase and deployed application
4. **Real-time Discussion** - Collaborate in the shared environment

### For Candidates  

1. **Join the Room** - Access the shared interview environment
2. **Present Your Work** - Walk through your actual project
3. **Live Demonstration** - Show features and explain architecture
4. **Answer in Context** - Discuss code with the actual implementation visible

### Technical Flow

1. **Repository Cloning** - GitHub repository is automatically cloned
2. **Docker Build** - Project is containerized using its Dockerfile
3. **Environment Deployment** - Container is deployed to cloud infrastructure
4. **WebSocket Connection** - Real-time terminal and UI sharing begins
5. **Collaborative Session** - Multiple users interact in shared environment

## 🔧 Configuration

### Environment Variables

Create `.env` file in the backend directory:

```bash
# Supabase Configuration
SUPABASE_URL=your_supabase_url
SUPABASE_ANON_KEY=your_supabase_anon_key

# AWS Configuration  
AWS_ACCESS_KEY_ID=your_aws_access_key
AWS_SECRET_ACCESS_KEY=your_aws_secret_key
AWS_REGION=your_aws_region
S3_BUCKET_NAME=your_s3_bucket

# Application Configuration
PORT=3009
FRONTEND_URL=http://localhost:3000
```

### Database Schema

The application uses Supabase with the following main tables:
- `running_rooms` - Active interview sessions
- `room_participants` - User participation tracking
- `terminal_outputs` - Real-time terminal logs

## 📋 Project Status

🚧 **Early Development** - We have a working prototype with:

✅ **Completed Features:**
- Live multiplayer environments
- Concurrent breakout rooms  
- GitHub repository cloning
- Docker deployment pipeline
- WebSocket real-time communication
- Basic UI/UX implementation

🔄 **In Progress:**
- Enhanced security and sandboxing
- Performance optimizations
- Extended language/framework support
- Advanced collaboration features

🎯 **Roadmap:**
- IDE-like code editing capabilities
- Voice/video integration
- Analytics and insights
- Enterprise features

## 📞 Contact

Built with passion by Sam and Martin.

**Questions or feedback?** We'd love to hear from you!

---

*"If you say you built something, you better be ready to back it up — because now it's right there on the screen."*
