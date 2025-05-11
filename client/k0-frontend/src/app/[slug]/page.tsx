import { createClient } from "@/utils/supabase/server";
import { cookies } from "next/headers";
import { GithubRepoInput } from "@/components/github-repo-input";
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

    return (
        <div className="flex flex-col items-center justify-center h-screen">
            <div>Room id: {roomId}</div>
            <GithubRepoInput />
        </div>
    );
}
