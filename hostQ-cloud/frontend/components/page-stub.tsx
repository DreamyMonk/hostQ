import { PageHead } from "./page-head";
import { Card, CardContent } from "./ui/card";
import { Construction } from "lucide-react";

// Placeholder used by routes that exist for layout completeness but whose
// content isn't built yet. Keeps the sidebar from 404-ing.
export function PageStub({ title, description }: { title: string; description?: string }) {
  return (
    <>
      <PageHead title={title} description={description} />
      <Card>
        <CardContent className="py-16 text-center">
          <Construction className="h-7 w-7 mx-auto text-faint mb-3" />
          <div className="text-sm font-medium">Under construction</div>
          <div className="text-xs text-muted mt-1">This module ships next.</div>
        </CardContent>
      </Card>
    </>
  );
}
