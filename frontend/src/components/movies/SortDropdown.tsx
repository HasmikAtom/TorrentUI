import { ArrowUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SORT_OPTIONS, type SortKey } from "./types";

type Props = {
  value: SortKey;
  onChange: (v: SortKey) => void;
};

export function SortDropdown({ value, onChange }: Props) {
  const current = SORT_OPTIONS.find((o) => o.key === value) ?? SORT_OPTIONS[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <ArrowUpDown className="mr-2 h-4 w-4" />
          {current.label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          value={value}
          onValueChange={(v) => onChange(v as SortKey)}
        >
          {SORT_OPTIONS.map((o) => (
            <DropdownMenuRadioItem key={o.key} value={o.key}>
              {o.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
