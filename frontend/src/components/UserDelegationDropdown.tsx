import { useState } from "react";
import { ChevronDown, User, UserPlus } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { User as UserType } from "@/lib/api/types";

interface UserDelegationDropdownProps {
  users: UserType[];
  isLoading?: boolean;
  onSelectUser: (userId: string) => void;
  disabled?: boolean;
  currentUserId?: string;
}

export function UserDelegationDropdown({ 
  users, 
  isLoading = false, 
  onSelectUser, 
  disabled = false,
  currentUserId 
}: UserDelegationDropdownProps) {
  const [open, setOpen] = useState(false);

  const handleSelectUser = (userId: string) => {
    onSelectUser(userId);
    setOpen(false);
  };

  // Filter out current user to avoid self-assignment
  const availableUsers = users.filter(user => user.id !== currentUserId);

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button 
          variant="outline" 
          size="sm" 
          disabled={disabled || isLoading}
          className="h-8 px-2"
        >
          <UserPlus className="h-3.5 w-3.5 mr-1" />
          Delegar
          <ChevronDown className="h-3.5 w-3.5 ml-1" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel>Selecionar usuário</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {isLoading ? (
          <DropdownMenuItem disabled>
            Carregando usuários...
          </DropdownMenuItem>
        ) : availableUsers.length === 0 ? (
          <DropdownMenuItem disabled>
            Nenhum usuário disponível
          </DropdownMenuItem>
        ) : (
          availableUsers.map((user) => (
            <DropdownMenuItem
              key={user.id}
              onClick={() => handleSelectUser(user.id)}
              className="cursor-pointer"
            >
              <div className="flex items-center space-x-2">
                <Avatar className="h-6 w-6">
                  <AvatarFallback className="text-xs">
                    {user.name ? user.name.charAt(0).toUpperCase() : 'U'}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col">
                  <span className="text-sm font-medium">{user.name || 'Sem nome'}</span>
                  <span className="text-xs text-muted-foreground">{user.email}</span>
                </div>
              </div>
            </DropdownMenuItem>
          ))
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
