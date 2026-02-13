# System Image ID Declaration

## Definitions

- `id`: local record ID in `system_images` table (internal primary key).
- `image_id`: upstream automation image ID (external identifier).

## Mandatory rule

When communicating with automation systems, always use `image_id`.

## Not interchangeable

- `id != image_id`
- `id` is only for local CRUD and local relations (such as `line_system_images.system_image_id`).
- `image_id` is the only identifier that should be sent to automation APIs (such as reset/reinstall template selection).

