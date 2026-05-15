import { LoaderCircle } from "lucide-react";

export function ButtonSpinner({
  className = "",
}: {
  className?: string;
}) {
  return <LoaderCircle size={15} className={`animate-spin ${className}`.trim()} />;
}

