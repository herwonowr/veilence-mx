export interface Workspace {
  id: string
  name: string
  slug: string
  description: string
  ownerId: string
  isActive: boolean
  role: string
  packageCount?: number | null
  memberCount?: number | null
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
  createdAt: string
  updatedAt: string
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


export interface Invitation {
  id: string
  workspaceId: string
  email: string
  roleId: string
  invitedBy: string
  status: string
  expiresAt: string
  createdAt: string
}

export interface InvitationInfo {
  email: string
  workspaceId: string
  expiresAt: string
  accepted: boolean
  expired: boolean
}

export interface MyInvitation {
  id: string
  workspaceId: string
  workspaceName: string
  email: string
  invitedByEmail: string
  status: string
  createdAt: string
  expiresAt: string
}

export interface AuditLogParams {
  action?: string
  resource?: string
  from_date?: string
  to_date?: string
  page?: number
  limit?: number
}

export interface AddMemberRequest {
  email: string
  firstName: string
  lastName: string
  roleId: string
  password?: string
}

export interface AddMemberResponse {
  member: WorkspaceMember
  userCreated: boolean
}
