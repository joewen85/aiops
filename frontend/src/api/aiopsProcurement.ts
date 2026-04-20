import { apiClient } from "@/api/client";
import type { ApiResponse } from "@/api/types";
import type {
  AIOpsProcurementExecutionResult,
  AIOpsProcurementIntent,
  AIOpsProcurementIntentRequest,
  AIOpsProcurementPlan,
  AIOpsProcurementProtocolSpec,
} from "@/types/aiopsProcurement";

export async function getAIOpsProcurementProtocol(): Promise<AIOpsProcurementProtocolSpec> {
  const { data } = await apiClient.get<ApiResponse<AIOpsProcurementProtocolSpec>>("/aiops/procurement/protocol");
  return data.data;
}

export async function parseAIOpsProcurementIntent(payload: AIOpsProcurementIntentRequest): Promise<{
  protocolVersion: string;
  intent: AIOpsProcurementIntent;
  clarifications: string[];
  next: string;
}> {
  const { data } = await apiClient.post<ApiResponse<{
    protocolVersion: string;
    intent: AIOpsProcurementIntent;
    clarifications: string[];
    next: string;
  }>>("/aiops/procurement/intents", payload);
  return data.data;
}

export async function buildAIOpsProcurementPlan(intent: AIOpsProcurementIntent): Promise<{
  protocolVersion: string;
  plan: AIOpsProcurementPlan;
  next: string;
}> {
  const { data } = await apiClient.post<ApiResponse<{
    protocolVersion: string;
    plan: AIOpsProcurementPlan;
    next: string;
  }>>("/aiops/procurement/plans", { intent });
  return data.data;
}

export async function executeAIOpsProcurementPlan(plan: AIOpsProcurementPlan, dryRun = true): Promise<{
  protocolVersion: string;
  result: AIOpsProcurementExecutionResult;
}> {
  const { data } = await apiClient.post<ApiResponse<{
    protocolVersion: string;
    result: AIOpsProcurementExecutionResult;
  }>>("/aiops/procurement/executions", { plan, dryRun });
  return data.data;
}
