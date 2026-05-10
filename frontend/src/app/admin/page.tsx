import { redirect } from "next/navigation"
import { ROUTES } from "@/core"

const AdminPage = () => redirect(ROUTES.ADMIN_USERS)

export default AdminPage
