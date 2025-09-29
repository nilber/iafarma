import { Bell, Search, User, ChevronDown, LogOut, Settings, UserIcon, Volume2, VolumeX, Bot, BotOff } from "lucide-react";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/contexts/AuthContext";
import { useSoundNotification } from "@/contexts/SoundNotificationContext";
import { useNavigate } from "react-router-dom";
import AICreditsDisplay from "./AICreditsDisplay";
import { GlobalSearch } from "@/components/ui/GlobalSearch";
import { useUnreadMessagesCount, useGlobalAISetting, useUpdateGlobalAISetting } from "@/lib/api/hooks";
import { useMemo, useCallback, memo } from "react";
// Debug hooks removidos

export function TopHeader() {
  const { user, logout } = useAuth();
  const { isEnabled, toggleSound } = useSoundNotification();
  const navigate = useNavigate();

  // Memoize isSystemAdmin to prevent unnecessary re-renders
  const isSystemAdmin = useMemo(() => user?.role === 'system_admin', [user?.role]);
  
  // Get unread messages count - only for tenant users, not system admin
  const { data: unreadData } = useUnreadMessagesCount({ enabled: !isSystemAdmin });
  const unreadCount = useMemo(() => 
    !isSystemAdmin && unreadData ? unreadData.unread_count || 0 : 0, 
    [isSystemAdmin, unreadData]
  );

  // Global AI setting - only for tenant users
  const { data: isAIGlobalEnabled = true } = useGlobalAISetting();
  const updateAISetting = useUpdateGlobalAISetting();

  // Debug hooks removidos para eliminar re-renders

  const handleToggleGlobalAI = useCallback(() => {
    updateAISetting.mutate(!isAIGlobalEnabled);
  }, [updateAISetting, isAIGlobalEnabled]);

  const handleLogout = useCallback(() => {
    logout();
  }, [logout]);

  const handleSettings = useCallback(() => {
    navigate('/settings');
  }, [navigate]);

  const handleProfile = () => {
    navigate('/profile');
  };

  return (
    <header className="h-16 border-b bg-card/50 backdrop-blur-sm flex items-center justify-between px-6 transition-all duration-150 ease-in-out">
      <div className="flex items-center gap-4">
        <SidebarTrigger className="text-muted-foreground hover:text-foreground" />
        
        {/* Global Search - Only show for tenant admins, not system admins */}
        {user?.role !== 'system_admin' && (
          <GlobalSearch />
        )}
      </div>

      <div className="flex items-center gap-4">
        {/* AI Credits Display - Only for non-admin users */}
        {user?.role !== 'system_admin' && <AICreditsDisplay />}

        {/* Sound Notifications Toggle */}
        <Button 
          variant="ghost" 
          size="icon" 
          onClick={toggleSound}
          className="relative"
          title={isEnabled ? "Desabilitar alertas sonoros" : "Habilitar alertas sonoros"}
        >
          {isEnabled ? (
            <Volume2 className="w-5 h-5" />
          ) : (
            <VolumeX className="w-5 h-5 text-muted-foreground" />
          )}
        </Button>

        {/* Global AI Toggle - Only for tenant users */}
        {!isSystemAdmin && (
          <Button 
            variant="ghost" 
            size="icon" 
            onClick={handleToggleGlobalAI}
            className={`relative transition-all duration-200 ${
              isAIGlobalEnabled 
                ? "text-green-600 hover:text-green-700" 
                : "text-red-500 hover:text-red-600"
            }`}
            title={isAIGlobalEnabled ? "Desabilitar IA globalmente" : "Habilitar IA globalmente"}
            disabled={updateAISetting.isPending}
          >
            {isAIGlobalEnabled ? (
              <Bot className="w-5 h-5" />
            ) : (
              <BotOff className="w-5 h-5" />
            )}
          </Button>
        )}

        {/* Notifications - Only show for tenant users with click to WhatsApp */}
        {!isSystemAdmin && (
          <Button 
            variant="ghost" 
            size="icon" 
            className="relative"
            onClick={() => navigate('/whatsapp')}
            title="Ver conversas do WhatsApp"
          >
            <Bell className="w-5 h-5" />
            {unreadCount > 0 && (
              <Badge className="absolute -top-1 -right-1 w-5 h-5 p-0 flex items-center justify-center bg-destructive text-destructive-foreground text-xs">
                {unreadCount > 99 ? '99+' : unreadCount}
              </Badge>
            )}
          </Button>
        )}

        {/* User Menu */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="flex items-center gap-2 h-10 px-3">
              <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center">
                <User className="w-4 h-4 text-primary-foreground" />
              </div>
              <div className="text-left hidden sm:block">
                <p className="text-sm font-medium">{user?.name || 'Usuário'}</p>
                <p className="text-xs text-muted-foreground capitalize">{user?.role || 'Usuário'}</p>
              </div>
              <ChevronDown className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel>Minha Conta</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleProfile}>
              <UserIcon className="w-4 h-4 mr-2" />
              Perfil
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleSettings}>
              <Settings className="w-4 h-4 mr-2" />
              Configurações
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={handleLogout}>
              <LogOut className="w-4 h-4 mr-2" />
              Sair
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}

// Memoize the component to prevent unnecessary re-renders
export default memo(TopHeader);