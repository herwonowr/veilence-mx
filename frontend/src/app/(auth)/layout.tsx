const AuthLayout = ({
  children,
}: {
  children: React.ReactNode
}) => {
  return (
    <div className="flex min-h-screen items-center justify-center py-8">
      {children}
    </div>
  )
}
export default AuthLayout
