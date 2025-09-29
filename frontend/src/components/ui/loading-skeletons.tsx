import { Skeleton } from "@/components/ui/skeleton";
import { TableRow, TableCell } from "@/components/ui/table";

interface TableSkeletonProps {
  rows?: number;
  columns?: number;
  includeAvatar?: boolean;
}

export function TableSkeleton({ 
  rows = 5, 
  columns = 4, 
  includeAvatar = false 
}: TableSkeletonProps) {
  return (
    <>
      {Array.from({ length: rows }).map((_, index) => (
        <TableRow key={`skeleton-${index}`}>
          {Array.from({ length: columns }).map((_, colIndex) => (
            <TableCell key={`skeleton-cell-${index}-${colIndex}`}>
              {colIndex === 0 && includeAvatar ? (
                <div className="flex items-center gap-3">
                  <Skeleton className="w-10 h-10 rounded-full" />
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-48" />
                  </div>
                </div>
              ) : (
                <Skeleton className="h-4 w-full" />
              )}
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  );
}

interface ListSkeletonProps {
  items?: number;
  showAvatar?: boolean;
}

export function ListSkeleton({ items = 3, showAvatar = false }: ListSkeletonProps) {
  return (
    <>
      {Array.from({ length: items }).map((_, index) => (
        <div key={`list-skeleton-${index}`} className="flex items-center space-x-4 p-4">
          {showAvatar && <Skeleton className="h-12 w-12 rounded-full" />}
          <div className="space-y-2 flex-1">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        </div>
      ))}
    </>
  );
}

interface CardSkeletonProps {
  showImage?: boolean;
}

export function CardSkeleton({ showImage = false }: CardSkeletonProps) {
  return (
    <div className="border rounded-lg p-4 space-y-3">
      {showImage && <Skeleton className="h-40 w-full" />}
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-4 w-1/2" />
      <div className="flex space-x-2">
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-20" />
      </div>
    </div>
  );
}
