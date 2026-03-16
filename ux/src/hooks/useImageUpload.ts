import { useState } from 'react';
import { uploadImage } from '../lib/api';

export function useImageUpload(projectId: string) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const upload = async (file: File): Promise<string | null> => {
    setUploading(true);
    setError(null);
    try {
      const { url } = await uploadImage(projectId, file);
      return url;
    } catch {
      setError('Image upload failed');
      return null;
    } finally {
      setUploading(false);
    }
  };

  return { upload, uploading, error };
}
