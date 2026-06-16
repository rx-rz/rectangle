import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/auth/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>

    <p>Welcome to Rectangle</p>

    
  </div>
}
