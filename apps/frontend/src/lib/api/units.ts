// Units API client
// Provides CRUD operations for unit management

import { api } from "./client";

// =============================================================================
// TYPES
// =============================================================================

export interface Unit {
    id: string;
    name: string;
    symbol: string;
    type: "distance" | "time" | "volume" | "count" | "custom";
    is_system: boolean;
    created_at?: string;
    updated_at?: string;
}

export interface UnitListResponse {
    items: Unit[];
    system_only?: boolean;
}

export interface CreateUnitRequest {
    name: string;
    symbol: string;
    type?: "distance" | "time" | "volume" | "count" | "custom";
}

export interface UpdateUnitRequest {
    name?: string;
    symbol?: string;
}

// =============================================================================
// API FUNCTIONS
// =============================================================================

/**
 * Get all units (system + user's custom units)
 */
export async function getUnits(systemOnly?: boolean): Promise<UnitListResponse> {
    const params = new URLSearchParams();
    if (systemOnly !== undefined) {
        params.set("system_only", String(systemOnly));
    }
    const query = params.toString();
    return api.get(`units${query ? `?${query}` : ""}`).json();
}

/**
 * Get a single unit by ID
 */
export async function getUnit(id: string): Promise<Unit> {
    return api.get(`units/${id}`).json();
}

/**
 * Create a custom unit
 */
export async function createUnit(data: CreateUnitRequest): Promise<Unit> {
    return api.post("units", { json: data }).json();
}

/**
 * Update a custom unit
 */
export async function updateUnit(id: string, data: UpdateUnitRequest): Promise<Unit> {
    return api.put(`units/${id}`, { json: data }).json();
}

/**
 * Delete a custom unit
 */
export async function deleteUnit(id: string): Promise<void> {
    await api.delete(`units/${id}`);
}
