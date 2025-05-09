import { createClient } from "@/utils/supabase/server";
import { cookies } from "next/headers";

export default async function Page({ params }: { params: { slug: string } }) {
    const { slug: roomId } = params;

    const cookieStore = cookies()
    const supabase = await createClient(cookieStore)

    const { data, error } = await supabase.from("running_rooms").select("*").eq("id", roomId)

    if (error) {
        return <div>Error: {error.message}</div>;
    }

    if (data.length === 0) {
        return <div>Room not found</div>;
    }

    return <div>Room id: {roomId}</div>;
}
