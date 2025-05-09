import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { ChevronRight } from 'lucide-react';
import { cookies } from "next/headers";
import { createClient } from "@/utils/supabase/server";
import { CreateNewRoomCard } from "@/components/create-room-cards";

export default async function Home() {
  
  const cookieStore = cookies()
  const supabase = createClient(cookieStore)
  
  return (
    <div className="flex h-screen w-screen items-center justify-center">
      <div className="flex gap-4">
        <CreateNewRoomCard />

      </div>
    </div>
  );
}
