/**
 * Permission constants — single source of truth for all frontend permission checks.
 * Must match the backend's permission model (user.permissions string array).
 * Any change here requires a corresponding backend update.
 */
export const Permissions = {
  /** Admin section access — dashboard, user/account management, system config */
  AdminView: 'admin:view',
} as const;

/** Check if a user has a specific permission. */
export function hasPermission(permissions: string[] | undefined, permission: string): boolean {
  return (permissions || []).includes(permission);
}

/** Check if a user has admin access. */
export function isAdmin(permissions: string[] | undefined): boolean {
  return hasPermission(permissions, Permissions.AdminView);
}
