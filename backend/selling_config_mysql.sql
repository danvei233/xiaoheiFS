-- Selling config export (MySQL compatible, explicit columns)
-- Source DB: backend/data/app.db
-- Generated at: 2026-02-11 21:25:28

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS=0;
START TRANSACTION;

DELETE FROM line_system_images;
DELETE FROM packages;
DELETE FROM system_images;
DELETE FROM billing_cycles;
DELETE FROM plan_groups;
DELETE FROM regions;
DELETE FROM goods_types;

-- TABLE: goods_types
INSERT INTO goods_types (id,code,name,active,sort_order,automation_category,automation_plugin_id,automation_instance_id,created_at,updated_at) VALUES(1,'lightboat_vps','����VPS',1,0,'automation','lightboat','default','2026-01-29 08:17:46','2026-01-29 08:17:46');

-- TABLE: regions
INSERT INTO regions (id,goods_type_id,code,name,active,created_at,updated_at) VALUES(1,1,'area-1','����',1,'2026-01-05 17:51:31','2026-01-05 17:51:31');
INSERT INTO regions (id,goods_type_id,code,name,active,created_at,updated_at) VALUES(2,1,'area-2','����',1,'2026-01-05 17:51:31','2026-01-08 15:41:49');

-- TABLE: plan_groups
INSERT INTO plan_groups (id,region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order,created_at,updated_at,goods_type_id) VALUES(1,1,'R620-2667v2',1,500.0,400.0,100.0,1000.0,0,0,1,0,0,1,0,0,1,0,0,1,1,1,-1,0,'2026-01-05 17:51:31','2026-01-10 17:30:09',1);
INSERT INTO plan_groups (id,region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order,created_at,updated_at,goods_type_id) VALUES(2,1,'RH2288-V3',3,400.0,400.0,100.0,1000.0,0,0,1,0,0,1,0,0,1,0,0,1,1,1,-1,1,'2026-01-05 17:51:31','2026-01-10 17:30:09',1);
INSERT INTO plan_groups (id,region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order,created_at,updated_at,goods_type_id) VALUES(3,1,'GMK-K6',4,800.0,600.0,100.0,1000.0,0,0,1,0,0,1,0,0,1,0,0,1,1,1,-1,2,'2026-01-05 17:51:31','2026-01-10 17:30:09',1);
INSERT INTO plan_groups (id,region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order,created_at,updated_at,goods_type_id) VALUES(4,1,'9950X',5,1200.0,800.0,100.0,1000.0,0,0,1,0,0,1,0,0,1,0,0,1,1,1,-1,3,'2026-01-05 17:51:31','2026-01-10 17:30:09',1);
INSERT INTO plan_groups (id,region_id,name,line_id,unit_core,unit_mem,unit_disk,unit_bw,add_core_min,add_core_max,add_core_step,add_mem_min,add_mem_max,add_mem_step,add_disk_min,add_disk_max,add_disk_step,add_bw_min,add_bw_max,add_bw_step,active,visible,capacity_remaining,sort_order,created_at,updated_at,goods_type_id) VALUES(5,2,'F-540',6,0.0,0.0,0.0,0.0,0,0,0,0,0,0,0,0,0,0,0,0,1,1,-1,0,'2026-01-05 18:41:20','2026-01-10 17:30:09',1);

-- TABLE: system_images
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(1,0,'Ubuntu 22.04','linux',1,'2026-01-05 17:51:31','2026-01-05 17:51:31');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(2,0,'Debian 12','linux',1,'2026-01-05 17:51:31','2026-01-05 17:51:31');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(3,0,'Windows Server 2022','windows',1,'2026-01-05 17:51:31','2026-01-05 17:51:31');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(4,35,'TEMP-LINUX','linux',1,'2026-01-05 18:41:21','2026-01-05 18:41:22');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(5,36,'CentOS 8 Stream','linux',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(6,37,'CentOS 7.9','linux',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(7,38,'Debian 12.5','linux',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(8,39,'Ubuntu 20.04','linux',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(9,40,'Ubuntu 22.04','linux',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(10,41,'Windows Server 2022 (Data Center Edition)','windows',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');
INSERT INTO system_images (id,image_id,name,type,enabled,created_at,updated_at) VALUES(11,42,'Windows 11','windows',1,'2026-01-05 18:41:21','2026-01-10 17:30:11');

-- TABLE: line_system_images
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(79,1,5,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(80,1,6,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(81,1,7,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(82,1,8,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(83,1,9,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(84,1,10,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(85,1,11,'2026-01-10 17:30:10');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(86,3,5,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(87,3,6,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(88,3,7,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(89,3,8,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(90,3,9,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(91,3,10,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(92,4,10,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(93,5,5,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(94,5,6,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(95,5,7,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(96,5,8,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(97,5,9,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(98,5,10,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(99,5,11,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(100,6,5,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(101,6,6,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(102,6,7,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(103,6,9,'2026-01-10 17:30:11');
INSERT INTO line_system_images (id,line_id,system_image_id,created_at) VALUES(104,6,10,'2026-01-10 17:30:11');

-- TABLE: billing_cycles
INSERT INTO billing_cycles (id,name,months,multiplier,min_qty,max_qty,active,sort_order,created_at,updated_at) VALUES(1,'monthly',1,1.0,1,24,1,1,'2026-01-05 17:51:31','2026-01-05 17:51:31');
INSERT INTO billing_cycles (id,name,months,multiplier,min_qty,max_qty,active,sort_order,created_at,updated_at) VALUES(2,'quarterly',3,2.8,1,12,1,2,'2026-01-05 17:51:31','2026-01-05 17:51:31');
INSERT INTO billing_cycles (id,name,months,multiplier,min_qty,max_qty,active,sort_order,created_at,updated_at) VALUES(3,'yearly',12,10.0,1,5,1,3,'2026-01-05 17:51:31','2026-01-05 17:51:31');

-- TABLE: packages
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(1,1,0,'2��4G 50G 10M 3.6GHz',2,4,40,10,'E5-2667 v2',1500.0,30,0,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(2,1,0,'4��8G 50G 10M 3.6GHz',4,8,40,10,'E5-2667 v2',2000.0,30,1,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(3,1,0,'4��12G 50G 15M 3.6GHz',4,12,40,15,'E5-2667 v2',2500.0,30,2,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(4,1,0,'6��16G 50G 15M 3.6GHz',6,16,40,15,'E5-2667 v2',3000.0,30,3,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(5,1,0,'8��24G 50G 20M 3.6GHz',8,24,40,20,'E5-2667 v2',4000.0,30,4,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(6,1,0,'8��32G 50G 20M 3.6GHz',8,32,40,20,'E5-2667 v2',7000.0,30,5,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(7,1,0,'10��36G 50G 20M 3.6GHz',10,36,40,20,'E5-2667 v2',8500.0,30,6,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(8,2,0,'4��4G',4,4,40,10,'E5-2697 v4',1000.0,30,0,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(9,2,0,'4��8G',4,8,40,10,'E5-2697 v4',1500.0,30,1,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(10,2,0,'6��12G',6,12,40,10,'E5-2697 v4',2200.0,30,2,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(11,2,0,'8��16G',8,16,40,10,'E5-2697 v4',3000.0,30,3,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(12,2,0,'12��24G',12,24,40,10,'E5-2697 v4',4000.0,30,4,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(13,2,0,'16��32G',16,32,40,10,'E5-2697 v4',7000.0,30,5,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(14,2,0,'16��36G',16,36,40,10,'E5-2697 v4',8000.0,30,6,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(15,3,0,'2��4G',2,4,40,10,'AMD R7 7840H',4000.0,30,0,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(16,3,0,'4��8G',4,8,40,10,'AMD R7 7840H',7000.0,30,1,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(17,3,0,'4��12G',4,12,40,10,'AMD R7 7840H',9000.0,30,2,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(18,3,0,'4��16G',4,16,40,10,'AMD R7 7840H',11000.0,30,3,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(19,3,0,'6��18G',6,18,40,10,'AMD R7 7840H',13000.0,30,4,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(20,3,0,'8��24G',8,24,40,10,'AMD R7 7840H',17000.0,30,5,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(21,3,0,'8��32G',8,32,40,10,'AMD R7 7840H',22000.0,30,6,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(22,4,0,'2��4G',2,4,40,10,'AMD R9 9950X',6000.0,30,0,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(23,4,0,'4��8G',4,8,40,10,'AMD R9 9950X',9000.0,30,1,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(24,4,0,'4��12G',4,12,40,10,'AMD R9 9950X',12000.0,30,2,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(25,4,0,'4��16G',4,16,40,10,'AMD R9 9950X',14000.0,30,3,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(26,4,0,'6��18G',6,18,40,10,'AMD R9 9950X',17000.0,30,4,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(27,4,0,'8��24G',8,24,40,10,'AMD R9 9950X',23000.0,30,5,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(28,4,0,'12��28G',12,28,40,10,'AMD R9 9950X',34000.0,30,6,1,1,-1,'2026-01-05 17:51:31','2026-01-05 17:51:31',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(29,1,1,'8c16g 100m',8,16,40,25,'',0.0,30,0,1,1,-1,'2026-01-05 18:41:20','2026-01-10 17:30:09',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(30,2,1,'8c16g 100m',8,16,40,25,'',0.0,30,0,1,1,-1,'2026-01-05 18:41:20','2026-01-10 17:30:09',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(31,3,1,'8c16g 100m',8,16,40,25,'',0.0,30,0,1,1,-1,'2026-01-05 18:41:20','2026-01-10 17:30:10',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(32,4,1,'8c16g 100m',8,16,40,25,'',0.0,30,0,1,1,-1,'2026-01-05 18:41:20','2026-01-10 17:30:10',1);
INSERT INTO packages (id,plan_group_id,product_id,name,cores,memory_gb,disk_gb,bandwidth_mbps,cpu_model,monthly_price,port_num,sort_order,active,visible,capacity_remaining,created_at,updated_at,goods_type_id) VALUES(33,5,1,'8c16g 100m',8,16,40,25,'',0.0,30,0,0,1,-1,'2026-01-05 18:41:20','2026-01-10 17:30:10',1);

COMMIT;
SET FOREIGN_KEY_CHECKS=1;

