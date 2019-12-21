/*
Navicat MySQL Data Transfer

Source Server         : localhost-root
Source Server Version : 50641
Source Host           : localhost:3306
Source Database       : taobao

Target Server Type    : MYSQL
Target Server Version : 50641
File Encoding         : 65001

Date: 2019-12-12 14:16:40
*/

SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for goods
-- ----------------------------
DROP TABLE IF EXISTS `goods`;
CREATE TABLE `goods`
(
    `id`             int(10) unsigned NOT NULL AUTO_INCREMENT,
    `created_at`     timestamp        NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`     timestamp        NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`     timestamp        NULL DEFAULT NULL,
    `good_name`      varchar(50)           DEFAULT NULL,
    `good_main_img`  varchar(255)          DEFAULT NULL,
    `good_desc_link` varchar(255)          DEFAULT NULL,
    `category_name`  varchar(255)          DEFAULT NULL,
    `taobaoke_link`  varchar(255)          DEFAULT NULL,
    `good_price`     double                DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_goodname` (`good_name`),
    KEY `idx_goods_deleted_at` (`deleted_at`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 9
  DEFAULT CHARSET = utf8;
