import { Link, useLocation } from "react-router-dom";
import { Home, Shield, Film, type LucideIcon } from "lucide-react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

type NavItem = {
  title: string;
  icon: LucideIcon;
  href: string;
};

const BASE_ITEMS: NavItem[] = [
  { title: "Home", icon: Home, href: "/" },
];

const MOVIES_ITEM: NavItem = { title: "Movies", icon: Film, href: "/movies" };

const ADMIN_ITEMS: NavItem[] = [
  { title: "Admin", icon: Shield, href: "/admin" },
];

export function NavMenu({ isAdmin, plexEnabled }: { isAdmin: boolean; plexEnabled: boolean }) {
  const location = useLocation();
  const items: NavItem[] = [...BASE_ITEMS];
  if (plexEnabled) items.push(MOVIES_ITEM);
  if (isAdmin) items.push(...ADMIN_ITEMS);

  return (
    <SidebarGroup>
      <SidebarGroupLabel>Navigation</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.title}>
              <SidebarMenuButton
                asChild
                isActive={location.pathname === item.href}
                tooltip={item.title}
              >
                <Link to={item.href}>
                  <item.icon />
                  <span>{item.title}</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
