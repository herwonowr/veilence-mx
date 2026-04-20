export interface Workspace {
  id: string
  name: string
  slug: string
  description: string
  ownerId: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface WorkspaceMember {
  id: string
  workspaceId: string
  userId: string
  roleId: string
  role: Role
  joinedAt: string
  email?: string
  firstName?: string
  lastName?: string
}

export interface Role {
  id: string
  workspaceId: string
  name: string
  description: string
  isSystem: boolean
  permissions?: Permission[]
}

export interface Permission {
  id: string
  resource: string
  action: string
}

export interface AuditLog {
  id: string
  userId: string
  workspaceId: string
  action: string
  resource: string
  resourceId: string
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
