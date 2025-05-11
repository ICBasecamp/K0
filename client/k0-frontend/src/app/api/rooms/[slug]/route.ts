import { createClient } from "@/utils/supabase/server";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";


export async function GET(request: Request, { params }: { params: { slug: string } }) {
  const { slug: roomId } = params;

  const cookieStore = cookies()
  const supabase = await createClient(cookieStore)

  const { data, error } = await supabase.from("running_rooms").select("*").eq("id", roomId)

  if (error) {
    return NextResponse.json({ error: error.message }, { status: 500 })
  }
  
  if (data.length === 0) {
      return NextResponse.json({ error: "Room not found" }, { status: 404 })
  }

  return NextResponse.json({ data })
}
