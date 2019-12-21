/*
Navicat MySQL Data Transfer

Source Server         : localhost-root
Source Server Version : 50641
Source Host           : localhost:3306
Source Database       : xlsxtomysql

Target Server Type    : MYSQL
Target Server Version : 50641
File Encoding         : 65001

Date: 2019-12-13 11:43:48
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for auth_assignment
-- ----------------------------
DROP TABLE IF EXISTS `auth_assignment`;
CREATE TABLE `auth_assignment` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(11) unsigned NOT NULL,
  `user_id` int(11) unsigned NOT NULL,
  `created_at` int(11) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_assignment_user_type` (`type`,`user_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4;

-- ----------------------------
-- Table structure for group
-- ----------------------------
DROP TABLE IF EXISTS `group`;
CREATE TABLE `group` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(30) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_group_name` (`name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4;

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(30) NOT NULL DEFAULT '',
  `auth_key` varchar(32) NOT NULL DEFAULT '',
  `password_hash` varchar(128) NOT NULL DEFAULT '',
  `group_id` int(11) unsigned DEFAULT NULL,
  `status` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `created_at` int(11) unsigned NOT NULL,
  `updated_at` int(11) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_idx_user_username` (`username`) USING BTREE,
  KEY `fk_user_group_id` (`group_id`),
  CONSTRAINT `fk_user_group_id` FOREIGN KEY (`group_id`) REFERENCES `group` (`id`) ON DELETE SET NULL ON UPDATE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4;
