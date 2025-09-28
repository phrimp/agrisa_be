import io
import logging
from datetime import timedelta
from typing import Optional, BinaryIO, Dict, Any
from minio import Minio
from minio.error import S3Error
from app.config.settings import get_settings

logger = logging.getLogger(__name__)


class MinIOClient:
    """MinIO client for satellite image storage operations."""

    def __init__(self):
        settings = get_settings()
        self.client = Minio(
            endpoint=settings.minio_endpoint,
            access_key=settings.minio_access_key,
            secret_key=settings.minio_secret_key,
            secure=settings.minio_secure,
        )
        self.bucket_name = settings.minio_bucket_name
        self._ensure_bucket_exists()

    def _ensure_bucket_exists(self):
        """Ensure the bucket exists, create if it doesn't."""
        try:
            if not self.client.bucket_exists(self.bucket_name):
                self.client.make_bucket(self.bucket_name)
                logger.info(f"Created MinIO bucket: {self.bucket_name}")
        except S3Error as e:
            logger.error(f"Error ensuring bucket exists: {e}")
            raise

    async def upload_file(
        self,
        file_path: str,
        file_data: BinaryIO,
        content_type: str = "application/octet-stream",
        metadata: Optional[Dict[str, str]] = None,
    ) -> bool:
        """
        Upload a file to MinIO.

        Args:
            file_path: Path within the bucket
            file_data: File data stream
            content_type: MIME type of the file
            metadata: Optional metadata dictionary

        Returns:
            bool: True if successful, False otherwise
        """
        try:
            # Get file size
            file_data.seek(0, 2)  # Seek to end
            file_size = file_data.tell()
            file_data.seek(0)  # Reset to beginning

            self.client.put_object(
                bucket_name=self.bucket_name,
                object_name=file_path,
                data=file_data,
                length=file_size,
                content_type=content_type,
                metadata=metadata or {},
            )
            logger.info(f"Successfully uploaded file: {file_path}")
            return True

        except S3Error as e:
            logger.error(f"Error uploading file {file_path}: {e}")
            return False

    async def download_file(self, file_path: str) -> Optional[bytes]:
        """
        Download a file from MinIO.

        Args:
            file_path: Path within the bucket

        Returns:
            File content as bytes, or None if error
        """
        try:
            response = self.client.get_object(self.bucket_name, file_path)
            data = response.read()
            response.close()
            response.release_conn()
            return data

        except S3Error as e:
            logger.error(f"Error downloading file {file_path}: {e}")
            return None

    async def get_file_info(self, file_path: str) -> Optional[Dict[str, Any]]:
        """
        Get file information from MinIO.

        Args:
            file_path: Path within the bucket

        Returns:
            Dictionary with file info, or None if error
        """
        try:
            stat = self.client.stat_object(self.bucket_name, file_path)
            return {
                "size": stat.size,
                "etag": stat.etag,
                "last_modified": stat.last_modified,
                "content_type": stat.content_type,
                "metadata": stat.metadata,
            }

        except S3Error as e:
            logger.error(f"Error getting file info {file_path}: {e}")
            return None

    async def delete_file(self, file_path: str) -> bool:
        """
        Delete a file from MinIO.

        Args:
            file_path: Path within the bucket

        Returns:
            bool: True if successful, False otherwise
        """
        try:
            self.client.remove_object(self.bucket_name, file_path)
            logger.info(f"Successfully deleted file: {file_path}")
            return True

        except S3Error as e:
            logger.error(f"Error deleting file {file_path}: {e}")
            return False

    async def file_exists(self, file_path: str) -> bool:
        """
        Check if a file exists in MinIO.

        Args:
            file_path: Path within the bucket

        Returns:
            bool: True if file exists, False otherwise
        """
        try:
            self.client.stat_object(self.bucket_name, file_path)
            return True
        except S3Error:
            return False

    def get_presigned_url(
        self, file_path: str, expires: timedelta = timedelta(hours=1)
    ) -> Optional[str]:
        """
        Generate a presigned URL for file access.

        Args:
            file_path: Path within the bucket
            expires: URL expiration time

        Returns:
            Presigned URL string, or None if error
        """
        try:
            url = self.client.presigned_get_object(
                bucket_name=self.bucket_name, object_name=file_path, expires=expires
            )
            return url

        except S3Error as e:
            logger.error(f"Error generating presigned URL for {file_path}: {e}")
            return None

    def get_upload_url(
        self, file_path: str, expires: timedelta = timedelta(hours=1)
    ) -> Optional[str]:
        """
        Generate a presigned URL for file upload.

        Args:
            file_path: Path within the bucket
            expires: URL expiration time

        Returns:
            Presigned upload URL, or None if error
        """
        try:
            url = self.client.presigned_put_object(
                bucket_name=self.bucket_name, object_name=file_path, expires=expires
            )
            return url

        except S3Error as e:
            logger.error(f"Error generating upload URL for {file_path}: {e}")
            return None

    async def list_files(self, prefix: str = "") -> list:
        """
        List files in the bucket with optional prefix.

        Args:
            prefix: Prefix to filter files

        Returns:
            List of file paths
        """
        try:
            objects = self.client.list_objects(
                bucket_name=self.bucket_name, prefix=prefix, recursive=True
            )
            return [obj.object_name for obj in objects]

        except S3Error as e:
            logger.error(f"Error listing files with prefix {prefix}: {e}")
            return []

    async def copy_file(self, source_path: str, dest_path: str) -> bool:
        """
        Copy a file within the bucket.

        Args:
            source_path: Source file path
            dest_path: Destination file path

        Returns:
            bool: True if successful, False otherwise
        """
        try:
            from minio.commonconfig import CopySource

            copy_source = CopySource(
                bucket_name=self.bucket_name, object_name=source_path
            )

            self.client.copy_object(
                bucket_name=self.bucket_name, object_name=dest_path, source=copy_source
            )
            logger.info(f"Successfully copied file: {source_path} -> {dest_path}")
            return True

        except S3Error as e:
            logger.error(f"Error copying file {source_path} to {dest_path}: {e}")
            return False


# Global MinIO client instance
minio_client = MinIOClient()

