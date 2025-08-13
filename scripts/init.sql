CREATE DATABASE IF NOT EXISTS `message_push` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `message_push`;

CREATE TABLE IF NOT EXISTS `t_message` (
  `message_id` VARCHAR(64) NOT NULL COMMENT '消息ID',
  `message_type` INT(11) NOT NULL COMMENT '消息类型',
  `content` TEXT NOT NULL COMMENT '消息内容',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`message_id`),
  KEY `idx_message_type` (`message_type`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB COMMENT='消息表';

CREATE TABLE IF NOT EXISTS `t_user_message` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
  `message_id` VARCHAR(64) NOT NULL COMMENT '消息表主键ID',
  `status` TINYINT(4) NOT NULL COMMENT '消息推送状态',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id_message_id` (`user_id`, `message_id`),
  KEY `idx_user_id_status_created_at` (`user_id`, `status`, `created_at`)
) ENGINE=InnoDB COMMENT='用户消息表';
