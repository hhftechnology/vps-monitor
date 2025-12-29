import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface ContainersPaginationProps {
  totalItems: number;
  startIndex: number;
  endIndex: number;
  page: number;
  totalPages: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

export function ContainersPagination({
  totalItems,
  startIndex,
  endIndex,
  page,
  totalPages,
  pageSize,
  onPageChange,
  onPageSizeChange,
}: ContainersPaginationProps) {
  const handlePageClick = (nextPage: number) => {
    if (nextPage < 1 || nextPage > totalPages || nextPage === page) {
      return;
    }
    onPageChange(nextPage);
  };

  const renderPaginationNumbers = () => {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, index) => {
        const pageNumber = index + 1;
        return (
          <PaginationItem key={pageNumber}>
            <PaginationLink
              href="#"
              onClick={(event) => {
                event.preventDefault();
                handlePageClick(pageNumber);
              }}
              isActive={pageNumber === page}
            >
              {pageNumber}
            </PaginationLink>
          </PaginationItem>
        );
      });
    }

    return (
      <>
        <PaginationItem>
          <PaginationLink
            href="#"
            onClick={(event) => {
              event.preventDefault();
              handlePageClick(1);
            }}
            isActive={page === 1}
          >
            1
          </PaginationLink>
        </PaginationItem>
        {page > 3 && (
          <PaginationItem>
            <PaginationEllipsis />
          </PaginationItem>
        )}
        {page > 2 && (
          <PaginationItem>
            <PaginationLink
              href="#"
              onClick={(event) => {
                event.preventDefault();
                handlePageClick(page - 1);
              }}
            >
              {page - 1}
            </PaginationLink>
          </PaginationItem>
        )}
        {page !== 1 && page !== totalPages && (
          <PaginationItem>
            <PaginationLink
              href="#"
              onClick={(event) => event.preventDefault()}
              isActive
            >
              {page}
            </PaginationLink>
          </PaginationItem>
        )}
        {page < totalPages - 1 && (
          <PaginationItem>
            <PaginationLink
              href="#"
              onClick={(event) => {
                event.preventDefault();
                handlePageClick(page + 1);
              }}
            >
              {page + 1}
            </PaginationLink>
          </PaginationItem>
        )}
        {page < totalPages - 2 && (
          <PaginationItem>
            <PaginationEllipsis />
          </PaginationItem>
        )}
        <PaginationItem>
          <PaginationLink
            href="#"
            onClick={(event) => {
              event.preventDefault();
              handlePageClick(totalPages);
            }}
            isActive={page === totalPages}
          >
            {totalPages}
          </PaginationLink>
        </PaginationItem>
      </>
    );
  };

  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="text-sm text-muted-foreground">
        {totalItems > 0 ? (
          <>
            Showing {startIndex}-{endIndex} of {totalItems}
          </>
        ) : (
          <>0 containers</>
        )}
      </div>
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Rows per page</span>
          <Select
            value={String(pageSize)}
            onValueChange={(value) => onPageSizeChange(Number(value))}
          >
            <SelectTrigger size="sm" className="w-[70px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">10</SelectItem>
              <SelectItem value="20">20</SelectItem>
              <SelectItem value="50">50</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Pagination className="mx-0 w-auto">
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                href="#"
                onClick={(event) => {
                  event.preventDefault();
                  handlePageClick(page - 1);
                }}
                className={
                  page === 1 || totalItems === 0
                    ? "pointer-events-none opacity-50"
                    : ""
                }
              />
            </PaginationItem>
            {renderPaginationNumbers()}
            <PaginationItem>
              <PaginationNext
                href="#"
                onClick={(event) => {
                  event.preventDefault();
                  handlePageClick(page + 1);
                }}
                className={
                  page === totalPages || totalItems === 0
                    ? "pointer-events-none opacity-50"
                    : ""
                }
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  );
}
