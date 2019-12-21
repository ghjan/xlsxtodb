DROP TABLE IF EXISTS `auth_assignment`;

CREATE TABLE `auth_assignment`
(
    `id`         int(11) unsigned NOT NULL AUTO_INCREMENT,
    `type`       int(11) unsigned NOT NULL,
    `user_id`    int(11) unsigned NOT NULL,
    `created_at` int(11) unsigned NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

DROP TABLE IF EXISTS `group`;

CREATE TABLE `group`
(
    `id`   int(11) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(50)      NOT NULL DEFAULT '',
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

LOCK TABLES `group` WRITE;

INSERT INTO `group` (`id`, `name`)
VALUES (1, '管理员'),
       (2, '用户');

UNLOCK TABLES;

SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user`
(
    `id`            int(11) unsigned    NOT NULL AUTO_INCREMENT,
    `username`      varchar(255)        NOT NULL DEFAULT '',
    `auth_key`      varchar(32)         NOT NULL DEFAULT '',
    `password_hash` varchar(128)        NOT NULL DEFAULT '',
    `group_id`      int(11) unsigned             DEFAULT NULL,
    `status`        tinyint(1) unsigned NOT NULL DEFAULT '0',
    `created_at`    int(11) unsigned    NOT NULL,
    `updated_at`    int(11) unsigned             DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `fk_user_group_id` (`group_id`),
    CONSTRAINT `fk_user_group_id` FOREIGN KEY (`group_id`) REFERENCES `group` (`id`) ON DELETE SET NULL ON UPDATE SET NULL
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;
