"use client";

import { ExternalLink } from "lucide-react";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { useLangFlowFlows, useLangFlowFlow } from "@/services/queries/use-langflow";

type LangFlowSelectorProps = {
  flowId: string;
  onChange: (flowId: string) => void;
};

export function LangFlowSelector({ flowId, onChange }: LangFlowSelectorProps) {
  const { data: flows, isLoading } = useLangFlowFlows();
  const { data: selectedFlow } = useLangFlowFlow(flowId);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  const langflowUrl = "http://localhost:7860"; // TODO: Make this configurable

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="flow-select">Select LangFlow Workflow</Label>
        <Select value={flowId} onValueChange={onChange}>
          <SelectTrigger id="flow-select">
            <SelectValue placeholder="Choose a workflow..." />
          </SelectTrigger>
          <SelectContent>
            {flows?.map((flow) => (
              <SelectItem key={flow.id} value={flow.id}>
                {flow.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {selectedFlow && (
        <Card>
          <CardHeader>
            <div className="flex items-start justify-between">
              <div className="space-y-1">
                <CardTitle className="text-sm">{selectedFlow.name}</CardTitle>
                <CardDescription>{selectedFlow.description || "No description available"}</CardDescription>
              </div>
              <Button
                variant="outline"
                size="sm"
                asChild
              >
                <a
                  href={`${langflowUrl}/flow/${selectedFlow.id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1"
                >
                  Edit in LangFlow
                  <ExternalLink className="h-3 w-3" />
                </a>
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              Last updated: {new Date(selectedFlow.updated_at).toLocaleString()}
            </p>
          </CardContent>
        </Card>
      )}

      {!flows || flows.length === 0 && !isLoading && (
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-muted-foreground text-center">
              No workflows found. Create one in{" "}
              <a
                href={langflowUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                LangFlow
              </a>
              .
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
