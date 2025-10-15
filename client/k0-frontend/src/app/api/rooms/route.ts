import { createClient } from "@/utils/supabase/server";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const generateRoomId = () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let result = '';
    for (let i = 0; i < 6; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  };



export async function POST(request: Request) {
  const cookieStore = cookies()
  const supabase = await createClient(cookieStore)

  const roomId = generateRoomId()

  const { data, error } = await supabase.from("running_rooms").insert({
    id: roomId,
  }).select();

  if (error) {
    return NextResponse.json({ error: error.message }, { status: 500 })
  }

  return NextResponse.json({ data })
  
}