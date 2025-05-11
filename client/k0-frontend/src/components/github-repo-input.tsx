"use client";

import { useState, useEffect } from "react";
import axios from "axios";

export const GithubRepoInput = () => {
    const [githubLinkText, setGithubLinkText] = useState("https://github.com/docker/example-voting-app");  
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const [logs, setLogs] = useState("");
    const [isContainerStarting, setIsContainerStarting] = useState(false);

    return (
        <div>
            <input type="text" value={githubLinkText} onChange={(e) => setGithubLinkText(e.target.value)} />
            <button onClick={() => {
                console.log(githubLinkText)
                setIsContainerStarting(true)
                axios.post(`${process.env.NEXT_PUBLIC_BACKEND_URL}/start-github-container`, {
                    room_id: "1",
                    github_link: githubLinkText
                })
                .then(res => {
                    setIsContainerStarting(false)
                    const wsConnectionName = res.data.ws_connection_name
                    console.log("Connecting to WebSocket with name:", wsConnectionName)
                    
                    const wsUrl = `ws://localhost:3009/ws/container-output/${wsConnectionName}`
                    console.log("WebSocket URL:", wsUrl)
                    
                    const ws = new WebSocket(wsUrl);
                    setSocket(ws);

                    ws.onopen = () => {
                        console.log("WebSocket connection opened successfully");
                    };

                    ws.onmessage = (e) => {
                        console.log(e.data)
                        setLogs(prevLogs => prevLogs + e.data + "\n")
                    };
                
                    ws.onclose = (event) => {
                        console.log("WebSocket disconnected:", event.code, event.reason)
                        setSocket(null)
                    };

                    ws.onerror = (error) => {
                        console.error("WebSocket error:", error)
                    };
                })
                .catch(err => {
                    console.log(""< err)
                })
            }}>Load Repository</button>
        </div>
    )
};
