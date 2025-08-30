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
  try {
    console.log("🚀 Starting room creation...")
    console.log("Environment check:", {
      supabaseUrl: process.env.SUPABASE_URL ? "✅ Set" : "❌ Missing",
      supabaseKey: process.env.SUPABASE_ANON_KEY ? "✅ Set" : "❌ Missing",
      nextPublicUrl: process.env.NEXT_PUBLIC_SUPABASE_URL ? "✅ Set" : "❌ Missing",
      nextPublicKey: process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY ? "✅ Set" : "❌ Missing"
    })
    
    console.log("Actual Supabase URL:", process.env.SUPABASE_URL)
    console.log("URL length:", process.env.SUPABASE_URL?.length)

    // Test basic HTTP connectivity to Supabase
    console.log("Testing basic connectivity to Supabase...")
    try {
      const testResponse = await fetch(`${process.env.SUPABASE_URL}/rest/v1/`, {
        method: 'GET',
        headers: {
          'apikey': process.env.SUPABASE_ANON_KEY!,
          'Authorization': `Bearer ${process.env.SUPABASE_ANON_KEY!}`
        }
      })
      console.log("Basic connectivity test status:", testResponse.status)
    } catch (connectError) {
      console.error("❌ Basic connectivity test failed:", connectError)
    }

    const cookieStore = cookies()
    const supabase = await createClient(cookieStore)

    const roomId = generateRoomId()
    console.log("Generated room ID:", roomId)

    console.log("Attempting to insert into running_rooms table...")
    const { data, error } = await supabase.from("running_rooms").insert({
      id: roomId,
    }).select();

    console.log("Supabase response:", { data, error })

    if (error) {
      console.error("❌ Supabase error:", error)
      return NextResponse.json({ 
        error: error.message,
        details: error,
        env_check: {
          supabaseUrl: !!process.env.SUPABASE_URL,
          supabaseKey: !!process.env.SUPABASE_ANON_KEY
        }
      }, { status: 500 })
    }

    console.log("✅ Room created successfully:", data)
    return NextResponse.json({ data })
    
  } catch (err) {
    console.error("❌ Unexpected error:", err)
    return NextResponse.json({ 
      error: "Internal server error", 
      details: err instanceof Error ? err.message : String(err) 
    }, { status: 500 })
  }
}