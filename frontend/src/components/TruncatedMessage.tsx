import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { MessageContent } from '@/components/ui/message-content';

interface TruncatedMessageProps {
  content: string;
  type?: string;
  mediaUrl?: string;
  mediaType?: string;
  filename?: string;
  className?: string;
  maxLength?: number;
}

export const TruncatedMessage: React.FC<TruncatedMessageProps> = ({
  content,
  type,
  mediaUrl,
  mediaType,
  filename,
  className,
  maxLength = 1000
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  
  const shouldTruncate = content && content.length > maxLength;
  const truncatedContent = shouldTruncate ? content.substring(0, maxLength) + '...' : content;

  if (!shouldTruncate) {
    return (
      <MessageContent 
        content={content}
        type={type}
        mediaUrl={mediaUrl}
        mediaType={mediaType}
        filename={filename}
        className={className}
      />
    );
  }

  return (
    <>
      <div>
        <MessageContent 
          content={truncatedContent}
          type={type}
          mediaUrl={mediaUrl}
          mediaType={mediaType}
          filename={filename}
          className={className}
        />
        <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
          <DialogTrigger asChild>
            <Button 
              variant="link" 
              size="sm" 
              className="mt-1 p-0 h-auto text-xs underline"
            >
              Ver mais
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Mensagem Completa</DialogTitle>
            </DialogHeader>
            <div className="mt-4">
              <MessageContent 
                content={content}
                type={type}
                mediaUrl={mediaUrl}
                mediaType={mediaType}
                filename={filename}
                className="prose dark:prose-invert max-w-none"
              />
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </>
  );
};
