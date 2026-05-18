use chrono::Duration;
use thiserror::Error;

use crate::domain::attachments::{
    build_download_link, build_storage_key, request_upload, AttachmentsError, DownloadLink,
    UploadRequest,
};

use super::ports::Clock;

#[derive(Debug, Error)]
pub enum AttachmentsServiceError {
    #[error(transparent)]
    Attachments(#[from] AttachmentsError),
}

pub struct AttachmentsService<C> {
    clock: C,
}

impl<C> AttachmentsService<C>
where
    C: Clock,
{
    pub fn new(clock: C) -> Self {
        Self { clock }
    }

    pub fn build_storage_key(
        &self,
        tenant_id: &str,
        attachable_type: &str,
        attachable_id: &str,
        file_name: &str,
    ) -> Result<String, AttachmentsServiceError> {
        build_storage_key(
            tenant_id,
            attachable_type,
            attachable_id,
            file_name,
            self.clock.now(),
        )
        .map_err(Into::into)
    }

    pub fn request_upload(
        &self,
        base_url: &str,
        storage_key: &str,
        ttl: Option<Duration>,
    ) -> Result<UploadRequest, AttachmentsServiceError> {
        request_upload(base_url, storage_key, ttl, self.clock.now()).map_err(Into::into)
    }

    pub fn build_download_link(
        &self,
        base_url: &str,
        id: &str,
        ttl: Option<Duration>,
    ) -> Result<DownloadLink, AttachmentsServiceError> {
        build_download_link(base_url, id, ttl, self.clock.now()).map_err(Into::into)
    }
}
