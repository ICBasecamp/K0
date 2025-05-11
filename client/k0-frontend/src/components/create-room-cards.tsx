"use client";

import { ChevronRight } from "lucide-react";
import { Card, CardContent, CardDescription, CardTitle } from "./ui/card";
import { useRouter } from "next/navigation";
import { Input } from "./ui/input";
import { Button } from "./ui/button";
import { useState } from "react";
import { cn } from "@/lib/utils";

export const CreateNewRoomCard = () => {
    const router = useRouter();
    return (
        <Card className="bg-neutral-900 border-neutral-800 py-6 px-3 cursor-pointer w-[320px]" onClick={() => {
            fetch("/api/rooms", {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
            })
            .then(res => res.json())
            .then(data => {
              console.log(data)
              router.push(`/${data.data[0].id}`)
            })
          }}>
            <CardContent className="flex flex-col h-full">
              <div className="flex flex-col flex-grow">
                <CardTitle className="text-xl font-bold text-white">
                  Create New Room
                </CardTitle>
                <CardDescription className="text-sm text-neutral-400">
                  Create a new room as an Interviewer and invite your candidates.
                </CardDescription>
              </div>
              <ChevronRight className="h-6 text-white mt-auto ml-auto" />
            </CardContent>
          </Card>
    )
};

export const JoinRoomCard = () => {

  const router = useRouter();

  const [roomId, setRoomId] = useState("");
  const [error, setError] = useState("");
  const handleJoinRoom = async () => {
    const response = await fetch(`/api/rooms/${roomId}`);
    if (response.ok) {
      router.push(`/${roomId}`);
    } else {
      setError("Room not found");
    }
  }

    return (
        <Card className="bg-neutral-900 border-neutral-800 py-6 px-3 w-[320px]">
            <CardContent >
                <CardTitle className="text-xl font-bold text-white">
                    Join Room
                </CardTitle>
                <CardDescription className="text-sm text-neutral-400">
                    Join a room by entering the room ID
                </CardDescription>
                {error && <p className="text-red-500">{error}</p>}
                <Input className="mt-2 text-white" type="text" placeholder="Room ID" value={roomId} onChange={(e) => setRoomId(e.target.value)} />
                <Button className={
                  cn(
                    "mt-4 float-right cursor-pointer bg-neutral-800 hover:bg-neutral-700 text-white",
                    roomId.length === 0 ? "disabled opacity-50 cursor-not-allowed" : "",
                  )
                } onClick={
                  () => {
                    if (roomId.length > 0) {
                      handleJoinRoom();
                    }
                  }
                }>Join</Button>
            </CardContent>
        </Card>
    )
}