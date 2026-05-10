import { z } from "zod"

const autoCreateDomainRefinement = (
  data: { autoCreateUser: boolean; allowedDomains?: string },
  ctx: z.RefinementCtx,
) => {
  if (data.autoCreateUser && (!data.allowedDomains || data.allowedDomains.trim() === "")) {
    ctx.addIssue({
      code: "custom",
      message: "Allowed domains are required when auto-create users is enabled",
      path: ["allowedDomains"],
    })
  }
}

const ssoConfigBaseSchema = z.object({
  displayName: z.string().min(1, "Display name is required"),
  isEnabled: z.boolean(),
  autoCreateUser: z.boolean(),
  allowedDomains: z.string().optional(),
}).superRefine(autoCreateDomainRefinement)

export const samlConfigSchema = ssoConfigBaseSchema.extend({
  provider: z.literal("saml"),
  samlEntityId: z.string().min(1, "IdP Entity ID is required"),
  samlSsoUrl: z.url("SSO URL must be a valid URL"),
  samlCertificate: z.string().min(1, "IdP certificate is required"),
  samlAttrEmail: z.string().min(1, "Email attribute is required"),
  samlAttrFirstName: z.string().min(1, "First name attribute is required"),
  samlAttrLastName: z.string().min(1, "Last name attribute is required"),
})

export const oauthConfigSchema = ssoConfigBaseSchema.extend({
  provider: z.enum(["google", "github"]),
  oauthClientId: z.string().min(1, "Client ID is required"),
  oauthClientSecret: z.string().min(1, "Client Secret is required"),
  googleHostedDomain: z.string().optional(),
  githubOrgs: z.string().optional(),
})

/** Schema for editing an existing config (provider already set, fields optional) */
const ssoConfigUpdateBaseSchema = z.object({
  displayName: z.string().min(1, "Display name is required"),
  isEnabled: z.boolean(),
  autoCreateUser: z.boolean(),
  allowedDomains: z.string().optional(),
}).superRefine(autoCreateDomainRefinement)

export const samlConfigUpdateSchema = ssoConfigUpdateBaseSchema.extend({
  samlEntityId: z.string().min(1, "IdP Entity ID is required"),
  samlSsoUrl: z.url("SSO URL must be a valid URL"),
  samlCertificate: z.string().min(1, "IdP certificate is required"),
  samlAttrEmail: z.string().min(1, "Email attribute is required"),
  samlAttrFirstName: z.string().min(1, "First name attribute is required"),
  samlAttrLastName: z.string().min(1, "Last name attribute is required"),
})

export const oauthConfigUpdateSchema = ssoConfigUpdateBaseSchema.extend({
  oauthClientId: z.string().min(1, "Client ID is required"),
  oauthClientSecret: z.string().optional(),
  googleHostedDomain: z.string().optional(),
  githubOrgs: z.string().optional(),
})
