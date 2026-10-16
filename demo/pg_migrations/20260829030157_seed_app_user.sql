-- +goose Up
-- 初始化数据：PG.app_user 的数据来源于 jsonplaceholder /users 接口，
-- 这里在初始化时一次性请求后硬编码写入，演示运行不再访问该接口。
INSERT INTO app_user (id, name, username, email, website) VALUES
    (1,  'Leanne Graham',            'Bret',              'Sincere@april.biz',          'hildegard.org'),
    (2,  'Ervin Howell',             'Antonette',         'Shanna@melissa.tv',          'anastasia.net'),
    (3,  'Clementine Bauch',         'Samantha',          'Nathan@yesenia.net',         'ramiro.info'),
    (4,  'Patricia Lebsack',         'Karianne',          'Julianne.OConner@kory.org',  'kale.biz'),
    (5,  'Chelsey Dietrich',         'Kamren',            'Lucio_Hettinger@annie.ca',   'demarco.info'),
    (6,  'Mrs. Dennis Schulist',     'Leopoldo_Corkery',  'Karley_Dach@jasper.info',    'ola.org'),
    (7,  'Kurtis Weissnat',          'Elwyn.Skiles',      'Telly.Hoeger@billy.biz',     'elvis.io'),
    (8,  'Nicholas Runolfsdottir V', 'Maxime_Nienow',     'Sherwood@rosamond.me',       'jacynthe.com'),
    (9,  'Glenna Reichert',          'Delphine',          'Chaim_McDermott@dana.io',    'conrad.com'),
    (10, 'Clementina DuBuque',       'Moriah.Stanton',    'Rey.Padberg@karina.biz',     'ambrose.net');

-- +goose Down
DELETE FROM app_user;
