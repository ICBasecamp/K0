import { CreateNewRoomCard, JoinRoomCard } from "@/components/create-room-cards";

export default async function Home() {
  
  return (
    <div className="relative flex h-screen w-screen items-center justify-center overflow-hidden bg-black">
      <div className="absolute h-[500px] w-[500px] rounded-full bg-red-500/30 blur-3xl opacity-40" style={{ top: '10%', left: '40%' }}></div>
      <div className="absolute h-[600px] w-[600px] rounded-full bg-blue-500/30 blur-3xl opacity-40" style={{ top: '20%', left: '50%' }}></div>
      <div className="relative flex flex-col items-center justify-center gap-16">
        <p className="text-5xl font-mono font-bold text-white max-w-2xl text-center">Evaluate engineers by what matters most: their real code.</p>
        <div className="flex gap-4">
          <CreateNewRoomCard />
          <JoinRoomCard />
        </div>
      </div>
    </div>
  );
}
