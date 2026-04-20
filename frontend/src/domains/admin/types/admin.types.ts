export interface Workspace {
  id: number
  name: string
  slug: string
  description: string
  ownerId: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface WorkspaceMember {
  id: number
  workspaceId: number
  userId: number
  roleId: number
  role: Role
  joinedAt: string
  email?: string
  firstName?: string
  lastName?: string
}

export interface Role {
  id: number
  workspaceId: number
  name: string
  description: string
  isSystem: boolean
  permissions?: Permission[]
}

export interface Permission {
  id: number
  resource: string
  action: string
}

export interface AuditLog {
  id: number
  userId: number
  workspaceId: number
  action: string
  resource: string
  resourceId: number
  details: string
  ipAddress: string
  userAgent: string
  correlationId: string
  createdAt: string
}


export interface AuditLogParams {
  action?: string
  resource?: string
  from_date?: string
  to_date?: string
  page?: number
  limit?: number
}
