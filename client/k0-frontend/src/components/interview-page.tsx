"use client";

import { useState } from "react";
import axios from "axios";
import { Terminal } from "./console";
import { Card, CardContent } from "./ui/card";
import { Input } from "./ui/input";

type Repository = {
    name: string;
    url: string;
    author: string;
    author_image_url: string;
}

const ImportRepository = ({ currentLogs, setLogs, setIsContainerStarting, isContainerStarting }: { 
    currentLogs: string, 
    setLogs: (logs: string) => void, 
    setIsContainerStarting: (isContainerStarting: boolean) => void, 
    isContainerStarting: boolean }) => {
    const [githubLinkText, setGithubLinkText] = useState("");  
    const [socket, setSocket] = useState<WebSocket | null>(null);
    const [currentRepository, setCurrentRepository] = useState<Repository | null>(null);

    return (
        <Card className="bg-neutral-900 rounded-lg shadow-lg border border-neutral-800 w-full max-w-2xl">
            <CardContent className="flex flex-col gap-4">
                <div className="flex flex-col gap-2">
                    <p className="text-white text-xl font-bold">Import Repository</p>
                    <p className="text-neutral-400 text-sm">Enter a project's GitHub repository URL to import it into the editor.</p>

                    <div className="flex flex-row gap-2">
                        <Input
                            type="text" 
                            value={githubLinkText} 
                            onChange={(e) => setGithubLinkText(e.target.value)}
                            className="flex-2 px-3 py-2 border rounded-lg focus:outline-none text-white"
                            placeholder="Enter GitHub repository URL"
                        />
                        <button 
                            onClick={async () => {
                                console.log(githubLinkText)
                                setIsContainerStarting(true)

                                try {
                                    const githubLinkTextSplit = githubLinkText.split("https://github.com/")
                                    if (githubLinkTextSplit.length !== 2) {
                                        throw new Error("Invalid GitHub repository URL")
                                    }
                                    const repoName = githubLinkTextSplit[1]
                                    const res = await axios.get(`https://api.github.com/repos/${repoName}`)
                                    setCurrentRepository({
                                        name: res.data.name,
                                        url: res.data.html_url,
                                        author: res.data.owner.login,
                                        author_image_url: res.data.owner.avatar_url
                                    })
                                } catch (err) {
                                    console.error("Error fetching repository:", err)
                                }

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

                                    ws.onmessage = (e: MessageEvent) => {
                                        
                                        // @ts-ignore
                                        setLogs(prevLogs => prevLogs + e.data + "\n")
                                    };
                                
                                    ws.onclose = (event: CloseEvent) => {
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
                            className="flex-1 cursor-pointer px-4 py-2 bg-neutral-800 text-white rounded-lg hover:bg-neutral-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
                        >
                            {isContainerStarting ? 'Loading...' : 'Import'}
                        </button>
                    </div>
                </div>
                {currentRepository && (
                    <div className="flex flex-col gap-2">
                        <p className="text-white text-xl font-bold">Current Repository</p>
                        <div className="flex flex-row gap-4 items-center">
                            <div className="flex flex-col gap-1">
                                <p className="text-neutral-400 text-sm">{currentRepository.name}</p>
                                <p className="text-neutral-400 text-sm">By {currentRepository.author}</p>
                            </div>
                            <img src={currentRepository.author_image_url} alt={currentRepository.author} className="h-8 w-8 rounded-lg" />
                        </div>
                    </div>
                )}
            </CardContent>
        </Card>
    )
    
};

export const Header = ({ roomId }: { roomId: string }) => {
    return (
        <div className="w-full bg-neutral-950 p-4 border-b border-neutral-800">
            <div className="max-w-7xl mx-auto flex items-center justify-between">
                <h1 className="text-white font-mono">Room: {roomId}</h1>
            </div>
        </div>
    );
};


export const InterviewPage = ({ roomId }: { roomId: string }) => {
    const [logs, setLogs] = useState("");
    const [isContainerStarting, setIsContainerStarting] = useState(false);

    return (
        <div className="flex flex-col h-screen w-screen">
            <Header roomId={roomId} />
            <div className="grid grid-cols-5 items-center justify-center h-full">
                <div className="col-span-2 flex flex-col h-full bg-neutral-950 border-r border-neutral-800">
                    <div className="flex flex-col p-8">
                        <ImportRepository currentLogs={logs} setLogs={setLogs} setIsContainerStarting={setIsContainerStarting} isContainerStarting={isContainerStarting} />
                    </div>
                </div>
                <div className="h-full col-span-3 p-4 bg-neutral-900">
                    <Terminal logs={logs} isLoading={isContainerStarting} />
                </div>
            </div>
        </div>
    )
};
