import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getImages } from "../api/get-images";
import { removeImage, type RemoveImageParams } from "../api/remove-image";

export function useImagesQuery() {
  return useQuery({
    queryKey: ["images"],
    queryFn: getImages,
    staleTime: 30_000,
  });
}

export function useRemoveImageMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: RemoveImageParams) => removeImage(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["images"] });
    },
  });
}
