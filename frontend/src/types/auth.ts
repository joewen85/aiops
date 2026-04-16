export interface UserProfile {
  id: number;
  username: string;
  displayName: string;
  email?: string;
  isActive: boolean;
  roles: string[];
  deptId?: string;
}

export interface LoginResponse {
  token: string;
  user: UserProfile;
}
