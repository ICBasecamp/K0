"use client";

import { ChevronRight } from "lucide-react";
import { Card, CardContent, CardDescription, CardTitle } from "./ui/card";
import { useRouter } from "next/navigation";

export const CreateNewRoomCard = () => {
    const router = useRouter();
    return (
        <Card className="bg-neutral-900 border-neutral-800 py-6 px-3 cursor-pointer" onClick={() => {
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
            <CardContent>
              <CardTitle className="text-xl font-bold text-white">
                Create New Room
              </CardTitle>
              <CardDescription className="text-sm text-neutral-400">
                Create a new room as an Interviewer.
              </CardDescription>
  
              <ChevronRight className="mt-8 ml-auto h-4 text-white" />
            </CardContent>
          </Card>
    )
};