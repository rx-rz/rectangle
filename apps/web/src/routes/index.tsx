import { Button } from '#/components/ui/button'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/')({ component: Home })

function Home() {
  return (
    <div className="h-screen">
      <h1 className="text-4xl font-bold">Welcome to TanStack Start</h1>
      <p className="mt-4 text-lg">
        Edit <code>src/routes/index.tsx</code> to get started.
      </p>
      <div className='mt-4 flex'>
        <Button variant={"default"} className='font-mono'>Primary</Button>
        <Button variant={"destructive"}>Primary</Button>
        <Button variant={"ghost"}>Primary</Button>
        <Button variant={"link"}>Primary</Button>

        <Button variant={"secondary"}>Primary</Button>
      </div>
    </div>
  )
}
