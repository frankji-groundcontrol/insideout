-- workshops 导出文件访问策略（仅允许成员下载）
ALTER TABLE storage.objects ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS juanleme_export_download ON storage.objects;
DROP POLICY IF EXISTS juanleme_export_upload ON storage.objects;
DROP POLICY IF EXISTS juanleme_export_delete ON storage.objects;

CREATE POLICY juanleme_export_download
ON storage.objects
FOR SELECT
TO authenticated
USING (
  bucket_id = 'workshops'
  AND name LIKE 'juanleme/exports/%'
  AND EXISTS (
    SELECT 1
    FROM juanleme.workspace_memberships wm
    WHERE wm.user_id = auth.uid()
      AND wm.workspace_id::text = split_part(storage.objects.name, '/', 3)
  )
);

-- 客户端禁止上传导出产物（仅 service role 可写）
CREATE POLICY juanleme_export_upload
ON storage.objects
FOR INSERT
TO authenticated
WITH CHECK (false);

-- 客户端禁止删除导出产物（仅 service role 可删）
CREATE POLICY juanleme_export_delete
ON storage.objects
FOR DELETE
TO authenticated
USING (false);
