"use client";

import { AlertCircle, CheckCircle2 } from "lucide-react";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useLangFlowHealth } from "@/services/queries/use-langflow";
import type { SessionType } from "@/types/api/sessions";

type SessionTypeSelectorProps = {
  value: SessionType;
  onChange: (type: SessionType) => void;
};

export function SessionTypeSelector({ value, onChange }: SessionTypeSelectorProps) {
  const { data: isLangFlowHealthy, isLoading } = useLangFlowHealth();

  return (
    <div className="space-y-4">
      <Label>Session Type</Label>
      <RadioGroup
        value={value}
        onValueChange={(v) => onChange(v as SessionType)}
        className="space-y-3"
      >
        <div className="flex items-start space-x-3 space-y-0 rounded-md border p-4">
          <RadioGroupItem value="claude-code" id="claude-code" className="mt-1" />
          <div className="flex-1 space-y-1">
            <Label
              htmlFor="claude-code"
              className="font-medium cursor-pointer"
            >
              Claude Code Agent (Markdown)
            </Label>
            <p className="text-sm text-muted-foreground">
              Traditional markdown-based agents with full repository access
            </p>
          </div>
        </div>

        <div className="flex items-start space-x-3 space-y-0 rounded-md border p-4">
          <RadioGroupItem
            value="langflow"
            id="langflow"
            disabled={!isLangFlowHealthy}
            className="mt-1"
          />
          <div className="flex-1 space-y-1">
            <div className="flex items-center gap-2">
              <Label
                htmlFor="langflow"
                className={`font-medium ${isLangFlowHealthy ? 'cursor-pointer' : 'cursor-not-allowed opacity-50'}`}
              >
                LangFlow Visual Workflow
              </Label>
              {!isLoading && (
                isLangFlowHealthy ? (
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                ) : (
                  <AlertCircle className="h-4 w-4 text-yellow-500" />
                )
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              Drag-and-drop workflow builder for visual AI orchestration
            </p>
          </div>
        </div>
      </RadioGroup>

      {!isLangFlowHealthy && !isLoading && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            LangFlow service is not available. Please contact your administrator.
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
