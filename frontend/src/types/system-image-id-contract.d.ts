/**
 * System image identifier contract:
 * - `id`: local DB primary key of system_images table.
 * - `image_id`: upstream automation image identifier.
 *
 * Always use `image_id` when communicating with automation systems
 * (for example reset/reinstall/template selection).
 */

export type SystemImageRecordID = number;
export type AutomationImageID = number;

export interface SystemImageIDContract {
  id: SystemImageRecordID;
  image_id: AutomationImageID;
}

