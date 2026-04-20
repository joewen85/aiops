export interface AIOpsProcurementProtocolSpec {
  protocolVersion: string;
  supportedActions: string[];
  supportedProviders: string[];
  supportedResourceTypes: string[];
  supportsDryRun: boolean;
  supportsApprovalFlow: boolean;
}

export interface AIOpsProcurementIntentRequest {
  requestId?: string;
  message: string;
  preferredProvider?: string;
  region?: string;
  quantity?: number;
  budgetLimit?: number;
  metadata?: Record<string, unknown>;
}

export interface AIOpsProcurementIntent {
  protocolVersion: string;
  intentId: string;
  action: string;
  provider: string;
  resourceType: string;
  region: string;
  quantity: number;
  budgetLimit: number;
  rawMessage: string;
  constraints?: Record<string, unknown>;
}

export interface AIOpsProcurementPlanStep {
  order: number;
  name: string;
  action: string;
  endpoint: string;
  parameters?: Record<string, unknown>;
}

export interface AIOpsProcurementPlan {
  protocolVersion: string;
  planId: string;
  intent: AIOpsProcurementIntent;
  estimatedCost: number;
  currency: string;
  requiresApproval: boolean;
  safetyChecks: string[];
  steps: AIOpsProcurementPlanStep[];
}

export interface AIOpsProcurementExecutionResult {
  protocolVersion: string;
  executionId: string;
  planId: string;
  status: string;
  summary?: Record<string, unknown>;
}
