"use client";

import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { LangFlowInput } from "@/types/api/sessions";

type LangFlowInputConfigProps = {
  value: LangFlowInput;
  onChange: (input: LangFlowInput) => void;
};

export function LangFlowInputConfig({ value, onChange }: LangFlowInputConfigProps) {
  const handleMessageChange = (message: string) => {
    onChange({ ...value, input_value: message });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Flow Input Configuration</CardTitle>
        <CardDescription>
          Configure inputs for the LangFlow workflow
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="flow-message">Message</Label>
          <Textarea
            id="flow-message"
            placeholder="Enter your message for the workflow..."
            value={(value.input_value as string) || ""}
            onChange={(e) => handleMessageChange(e.target.value)}
            rows={4}
          />
          <p className="text-sm text-muted-foreground">
            This message will be passed to the LangFlow workflow as input
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
