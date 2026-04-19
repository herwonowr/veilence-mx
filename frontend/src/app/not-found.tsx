import Link from "next/link"
import { Button, Card, CardContent } from "@/ui"
import { FileQuestion } from "lucide-react"

const NotFound = () => {
  return (
    <div className="flex min-h-[50vh] items-center justify-center p-4">
      <Card className="max-w-md w-full">
        <CardContent className="pt-6 text-center space-y-4">
          <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted">
            <FileQuestion className="size-6 text-muted-foreground" />
          </div>
          <h2 className="text-xl font-semibold">Page not found</h2>
          <p className="text-sm text-muted-foreground">
            The page you are looking for does not exist or has been moved.
          </p>
          <Link href="/">
            <Button className="w-full">Go to Dashboard</Button>
          </Link>
        </CardContent>
      </Card>
    </div>
  )
}
export default NotFound
