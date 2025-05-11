"use client";

import { useState, useEffect } from "react";
import axios from "axios";

const Terminal = ({ logs, isLoading }: { logs: string, isLoading: boolean }) => {
    return (
        <div className="bg-black text-green-400 p-4 rounded-lg font-mono h-[400px] overflow-y-auto whitespace-pre-wrap">
            {isLoading ? (
                <div className="flex items-center space-x-2">
                    <div className="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-green-400"></div>
                    <span>Starting container...</span>
                </div>
            ) : (
                logs || <span className="text-gray-500">No logs available</span>
            )}
        </div>
    );
};

export const Console = () => {
    const [githubLinkText, setGithubLinkText] = useState("");  
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const [logs, setLogs] = useState("");
    const [isContainerStarting, setIsContainerStarting] = useState(false);

    return (
        <div className="space-y-4 p-4">
            <div className="flex space-x-2">
                <input 
                    type="text" 
                    value={githubLinkText} 
                    onChange={(e) => setGithubLinkText(e.target.value)}
                    className="flex-1 px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Enter GitHub repository URL"
                />
                <button 
                    onClick={() => {
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
                            setIsContainerStarting(false)
                            console.log(""< err)
                        })
                    }}
                    disabled={isContainerStarting}
                    className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 disabled:cursor-not-allowed"
                >
                    {isContainerStarting ? 'Loading...' : 'Load Repository'}
                </button>
            </div>
            <Terminal logs={logs} isLoading={isContainerStarting} />
        </div>
    )
};
