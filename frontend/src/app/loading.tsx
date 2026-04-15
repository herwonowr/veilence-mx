import { Loader2 } from "lucide-react"

const Loading = () => {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <Loader2 className="size-8 animate-spin text-muted-foreground" />
    </div>
  )
}
export default Loading
