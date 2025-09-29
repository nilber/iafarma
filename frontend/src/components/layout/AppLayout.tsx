import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "./AppSidebar";
import TopHeader from "./TopHeader";
import { useWebSocket } from "@/hooks/useWebSocket";
import { useUnreadMessagesCount } from "@/lib/api/hooks";
import { useState, useEffect } from "react";
import { useAuth } from "@/contexts/AuthContext";

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const { user } = useAuth();
  const isSystemAdmin = user?.role === 'system_admin';

  // Initialize WebSocket connection
  useWebSocket();

  // Get unread messages count - only for tenant users, not system admin
  const { data: unreadData } = useUnreadMessagesCount({ enabled: !isSystemAdmin });
  const unreadCount = !isSystemAdmin && unreadData ? unreadData.unread_count || 0 : 0;

  // Update document title with unread count - only for tenant users
  useEffect(() => {
    const baseTitle = "IAFarma";
    if (!isSystemAdmin && unreadCount > 0) {
      document.title = `(${unreadCount}) ${baseTitle}`;
    } else {
      document.title = baseTitle;
    }
  }, [unreadCount, isSystemAdmin]);

  return (
    <SidebarProvider>
      <div className="min-h-screen flex w-full bg-gradient-subtle">
        <AppSidebar />
        <div className="flex-1 flex flex-col">
          <TopHeader />
          <main className="flex-1 p-6">
            {children}
          </main>
        </div>
      </div>
      {/* Debug components removidos para eliminar re-renders */}
    </SidebarProvider>
  );
}