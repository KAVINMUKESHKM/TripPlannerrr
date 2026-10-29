import type { Attraction, NearbyRestaurant, RouteSummary, SearchDay, Trip, TripInput, User } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}/api/v1${path}`, {
    ...init,
    credentials: "include",
    headers: { ...(init?.body ? { "Content-Type": "application/json" } : {}), ...init?.headers },
    cache: "no-store",
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as { error?: string; message?: string };
    throw new ApiError(body.error || body.message || `Request failed (${response.status})`, response.status);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}

function numeric(value: unknown): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function normalizeAttraction(raw: Record<string, unknown>): Attraction | null {
  const candidate = (raw.attraction as Record<string, unknown> | undefined) ?? raw;
  const id = numeric(candidate.id as number | string);
  const name = String(candidate.name ?? "").trim();
  if (!id || !name) return null;
  return {
    id,
    name,
    description: String(candidate.description ?? ""),
    category: String(candidate.category ?? "Place"),
    best_climate: typeof candidate.best_climate === "string" ? candidate.best_climate : undefined,
    ideal_weather: typeof candidate.ideal_weather === "string" ? candidate.ideal_weather : undefined,
    ideal_temp_min: candidate.ideal_temp_min == null ? undefined : numeric(candidate.ideal_temp_min as number | string),
    ideal_temp_max: candidate.ideal_temp_max == null ? undefined : numeric(candidate.ideal_temp_max as number | string),
    latitude: numeric(candidate.latitude as number | string),
    longitude: numeric(candidate.longitude as number | string),
    source_url: typeof candidate.source_url === "string" ? candidate.source_url : undefined,
    image: typeof candidate.image === "string" ? candidate.image : undefined,
    address: typeof candidate.address === "string" ? candidate.address : undefined,
    rating: candidate.rating == null ? undefined : numeric(candidate.rating as number | string),
  };
}

export async function searchAttractions(input: { destination: string; days: number; date: string; category?: string }): Promise<SearchDay[]> {
  const query = new URLSearchParams({ destination: input.destination, days: String(input.days), date: input.date });
  if (input.category) query.set("category", input.category);
  const result = await request<unknown>(`/attractions?${query.toString()}`);
  if (!Array.isArray(result)) throw new ApiError("The attractions response had an unexpected format.", 502);

  const grouped = new Map<number, SearchDay>();
  for (const rawDay of result as Array<Record<string, unknown>>) {
    const dayNumber = numeric(rawDay.day) || 1;
    const day: SearchDay = grouped.get(dayNumber) ?? {
      day: dayNumber,
      date: typeof rawDay.date === "string" ? rawDay.date : undefined,
      weather: typeof rawDay.weather === "string" ? rawDay.weather : undefined,
      temperature: rawDay.temperature == null ? undefined : numeric(rawDay.temperature as number | string),
      attractions: [],
    };
    const sources: unknown[] = Array.isArray(rawDay.attractions)
      ? rawDay.attractions
      : Array.isArray(rawDay.suggestions)
        ? (rawDay.suggestions as Array<Record<string, unknown>>).flatMap((suggestion) => Array.isArray(suggestion.attractions) ? suggestion.attractions : [])
        : [];
    for (const source of sources) {
      if (source && typeof source === "object") {
        const place = normalizeAttraction(source as Record<string, unknown>);
        if (place && !day.attractions.some((existing) => existing.id === place.id)) day.attractions.push(place);
      }
    }
    grouped.set(dayNumber, day);
  }
  return Array.from(grouped.values()).sort((a, b) => a.day - b.day);
}

export const register = (input: { email: string; password: string; full_name: string }) =>
  request<User>("/auth/register", { method: "POST", body: JSON.stringify(input) });
export const login = (input: { email: string; password: string }) =>
  request<User>("/auth/login", { method: "POST", body: JSON.stringify(input) });
export const logout = () => request<void>("/auth/logout", { method: "POST" });
export const logoutAll = () => request<void>("/auth/logout-all", { method: "POST" });
export const requestPasswordReset = (email: string) => request<{ message: string }>("/auth/password/forgot", { method: "POST", body: JSON.stringify({ email }) });
export const resetPassword = (token: string, password: string) => request<void>("/auth/password/reset", { method: "POST", body: JSON.stringify({ token, password }) });
export const currentUser = () => request<User>("/auth/me");
export const googleSignInURL = `${API_BASE}/api/v1/auth/google`;

export const listTrips = () => request<Trip[]>("/trips");
export const getTrip = (id: string | number) => request<Trip>(`/trips/${id}`);
export const getAttraction = (id: string | number) => request<Attraction>(`/attractions/${id}`);
export const getNearbyRestaurants = (id: string | number) => request<NearbyRestaurant[]>(`/attractions/${id}/restaurants`);
export const createTrip = (input: TripInput) => request<Trip>("/trips", { method: "POST", body: JSON.stringify(input) });
export const updateTrip = (id: string | number, input: TripInput) => request<Trip>(`/trips/${id}`, { method: "PUT", body: JSON.stringify(input) });
export const deleteTrip = (id: string | number) => request<void>(`/trips/${id}`, { method: "DELETE" });
export const getRoute = (waypoints: Array<{ latitude: number; longitude: number }>) =>
  request<RouteSummary>("/routes", { method: "POST", body: JSON.stringify({ waypoints }) });

export function findAttraction(tripDays: SearchDay[], id: number): Attraction | undefined {
  return tripDays.flatMap((day) => day.attractions).find((place) => place.id === id);
}
