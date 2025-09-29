import { useState, useEffect } from "react";
import { 
  BarChart3, 
  MessageSquare, 
  ShoppingCart, 
  Users, 
  Package, 
  Settings,
  Building2,
  Home,
  QrCode,
  TrendingUp,
  Phone,
  AlertTriangle,
  Mail,
  Bot,
  Crown,
  Ticket,
  Layers,
  Brain,
  Database,
  CreditCard,
  Shield,
  Zap,
  Calendar,
  User,
  Wrench,
  Puzzle
} from "lucide-react";
import { NavLink, useLocation } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { useUnreadMessagesCount } from "@/lib/api/hooks";
import { useTenant } from "@/hooks/useTenant";
import { apiClient } from "@/lib/api/client";
import { Tenant } from "@/lib/api/types";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarHeader,
  useSidebar,
} from "@/components/ui/sidebar";

const mainItems = [
  { title: "Dashboard", url: "/", icon: Home },
  { title: "WhatsApp", url: "/whatsapp", icon: MessageSquare },
  { title: "Vendas", url: "/sales", icon: ShoppingCart },
  { title: "Produtos", url: "/products", icon: Package },
  { title: "Categorias", url: "/categories", icon: Layers },
  { title: "Clientes", url: "/customers", icon: Users },
  { title: "Relatórios", url: "/reports", icon: BarChart3 },
];

const adminItems = [
  { title: "Empresas", url: "/admin/tenants", icon: Building2 },
  // { title: "Administrar Canais", url: "/admin/channels", icon: Phone },
  { title: "Planos", url: "/admin/plans", icon: Crown },
  { title: "Logs de Erro", url: "/admin/error-logs", icon: AlertTriangle },
];

const whatsappItems = [
  { title: "Conversas", url: "/whatsapp/conversations", icon: MessageSquare },
  { title: "Canais e Alertas", url: "/whatsapp/connection", icon: QrCode },
];

const salesItems = [
  { title: "Pedidos", url: "/sales/orders", icon: ShoppingCart },
  { title: "Dashboard", url: "/sales/dashboard", icon: TrendingUp },
  { title: "Formas de Pagamento", url: "/sales/payment-methods", icon: CreditCard },
];

const productsItems = [
  { title: "Produtos", url: "/products", icon: Package },
  { title: "Categorias", url: "/categories", icon: Layers },
];




export function AppSidebar() {
  const location = useLocation();
  const { user } = useAuth();
  const { tenant, isSales } = useTenant();
  const currentPath = location.pathname;

  const isActive = (path: string) => {
    if (path === "/") return currentPath === "/";
    
    // Special case: /categories should be considered as part of /products
    if (path === "/products" && currentPath.startsWith("/categories")) {
      return true;
    }
    
    return currentPath.startsWith(path);
  };

  const getNavCls = (path: string) => {
    const active = isActive(path);
    return active 
      ? "bg-primary text-primary-foreground font-medium shadow-custom-sm" 
      : "hover:bg-accent text-muted-foreground hover:text-accent-foreground transition-all duration-200";
  };

  const getSubNavCls = (path: string) => {
    // For submenu items, only highlight if the path matches exactly
    const active = currentPath === path;
    return active 
      ? "bg-primary text-primary-foreground font-medium shadow-custom-sm" 
      : "hover:bg-accent text-muted-foreground hover:text-accent-foreground transition-all duration-200";
  };

  // Check if user is system admin
  const isSystemAdmin = user?.role === 'system_admin';
  
  // Get unread messages count - only for tenant users, not system admin
  const { data: unreadData } = useUnreadMessagesCount({ enabled: !isSystemAdmin });
  const unreadCount = !isSystemAdmin && unreadData ? unreadData.unread_count || 0 : 0;
  
  // Define items based on user role and business type
  const getVisibleMainItems = () => {
    if (isSystemAdmin) {
      // System admin only sees Dashboard and Empresas
      return [
        { title: "Dashboard", url: "/", icon: Home },
      ];
    }
    
    // Start with base items
    let items = [
      { title: "Dashboard", url: "/", icon: Home },
      { title: "WhatsApp", url: "/whatsapp", icon: MessageSquare },
    ];

    // Add unread count to WhatsApp if available
    if (unreadCount > 0) {
      items[1] = {
        ...items[1],
        title: `WhatsApp (${unreadCount > 99 ? '99+' : unreadCount})`
      };
    }

    // Sales profile (default): show all items
    items.push(
      { title: "Vendas", url: "/sales", icon: ShoppingCart },
      { title: "Produtos", url: "/products", icon: Package },
      { title: "Clientes", url: "/customers", icon: Users },      
      { title: "Relatórios", url: "/reports", icon: BarChart3 }
    );    

    return items;
  };

  return (
    <Sidebar
      className="w-64 border-r bg-card/50 backdrop-blur-sm"
    >
      <SidebarHeader className="border-b p-4">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-gradient-primary rounded-lg flex items-center justify-center">
            <MessageSquare className="w-4 h-4 text-primary-foreground" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-foreground">IAFarma</h2>
            <p className="text-xs text-muted-foreground">Sales & Support</p>
          </div>
        </div>
      </SidebarHeader>

      <SidebarContent className="py-4">
        <SidebarGroup>
          <SidebarGroupLabel>
            Principal
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {getVisibleMainItems().map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild className="h-11">
                    <NavLink to={item.url} className={getNavCls(item.url)}>
                      <item.icon className="w-5 h-5 shrink-0" />
                      <span className="ml-3">{item.title}</span>
                    </NavLink>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* WhatsApp Submenu - Only for tenant users */}
        {!isSystemAdmin && isActive("/whatsapp") && (
          <SidebarGroup className="bg-muted/20 border-l-2 border-muted-foreground/20 ml-2 rounded-md">
            <SidebarGroupLabel className="text-muted-foreground font-medium text-xs">WhatsApp</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {whatsappItems.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild className="h-10">
                      <NavLink to={item.url} className={getSubNavCls(item.url)}>
                        <item.icon className="w-4 h-4 shrink-0" />
                        <span className="ml-3">{item.title}</span>
                      </NavLink>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        {/* Products Submenu - Only for sales business type */}
        {!isSystemAdmin && isSales && (isActive("/products") || isActive("/categories")) && (
          <SidebarGroup className="bg-muted/20 border-l-2 border-muted-foreground/20 ml-2 rounded-md">
            <SidebarGroupLabel className="text-muted-foreground font-medium text-xs">Produtos</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {productsItems.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild className="h-10">
                      <NavLink to={item.url} className={getSubNavCls(item.url)}>
                        <item.icon className="w-4 h-4 shrink-0" />
                        <span className="ml-3">{item.title}</span>
                      </NavLink>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        {/* Sales Submenu - Only for sales business type */}
        {!isSystemAdmin && isSales && isActive("/sales") && (
          <SidebarGroup className="bg-muted/20 border-l-2 border-muted-foreground/20 ml-2 rounded-md">
            <SidebarGroupLabel className="text-muted-foreground font-medium text-xs">Vendas</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {salesItems.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild className="h-10">
                      <NavLink to={item.url} className={getSubNavCls(item.url)}>
                        <item.icon className="w-4 w-4 shrink-0" />
                        <span className="ml-3">{item.title}</span>
                      </NavLink>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

       
        {/* Admin Section - Only for System Admins */}
        {isSystemAdmin && (
          <SidebarGroup>
            <SidebarGroupLabel>
              Administração
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {adminItems.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild className="h-11">
                      <NavLink to={item.url} className={getNavCls(item.url)}>
                        <item.icon className="w-5 h-5 shrink-0" />
                        <span className="ml-3">{item.title}</span>
                      </NavLink>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
       
        {/* Regular Settings - For all users */}
        {!isSystemAdmin && (
          <SidebarGroup>
            <SidebarGroupLabel>
              Configurações
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild className="h-11">
                    <NavLink to="/settings" className={getNavCls("/settings")}>
                      <Settings className="w-5 h-5 shrink-0" />
                      <span className="ml-3">Configurações</span>
                    </NavLink>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
      </SidebarContent>
    </Sidebar>
  );
}