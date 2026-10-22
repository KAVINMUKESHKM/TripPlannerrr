import { currentUser } from "./api";
import type { User } from "./types";

export async function getSessionUser(): Promise<User | null> {
  try {
    return await currentUser();
  } catch {
    return null;
  }
}
