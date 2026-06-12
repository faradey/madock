package db

import "testing"

func TestParseMagentoEnv(t *testing.T) {
	content := `<?php
return [
    'db' => [
        'connection' => [
            'default' => [
                'host' => 'db.internal:3307',
                'dbname' => 'magento',
                'username' => 'mguser',
                'password' => 'mgp@ss',
                'active' => '1',
            ],
            'indexer' => [
                'host' => 'other-host',
                'dbname' => 'other-db',
                'username' => 'other-user',
                'password' => 'other-pass',
            ],
        ],
    ],
];`
	got := parseMagentoEnv(content)
	want := `{"host":"db.internal","dbname":"magento","username":"mguser","password":"mgp@ss","port":"3307"}`
	if got != want {
		t.Fatalf("parseMagentoEnv\n got: %s\nwant: %s", got, want)
	}
}

func TestParseDatabaseURL(t *testing.T) {
	cases := []struct {
		name, content, want string
	}{
		{
			name:    "shopware mysql",
			content: "APP_ENV=prod\nDATABASE_URL=\"mysql://swuser:swpass@127.0.0.1:3306/shopware\"\n",
			want:    `{"host":"127.0.0.1","dbname":"shopware","username":"swuser","password":"swpass","port":"3306"}`,
		},
		{
			name:    "postgres no port, url-encoded password",
			content: "DATABASE_URL=postgresql://pguser:p%40ss%3Aword@pg.example.com/saleor?sslmode=disable\n",
			want:    `{"host":"pg.example.com","dbname":"saleor","username":"pguser","password":"p@ss:word","port":""}`,
		},
		{
			name:    "env.local overrides env",
			content: "DATABASE_URL=mysql://local:local@local-host:3306/localdb\nDATABASE_URL=mysql://prod:prod@prod-host:3306/proddb\n",
			want:    `{"host":"local-host","dbname":"localdb","username":"local","password":"local","port":"3306"}`,
		},
		{
			name:    "export prefix",
			content: "export DATABASE_URL='mysql://u:p@h:3306/d'\n",
			want:    `{"host":"h","dbname":"d","username":"u","password":"p","port":"3306"}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseDatabaseURL(c.content); got != c.want {
				t.Fatalf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}

func TestParseWpConfig(t *testing.T) {
	content := `<?php
define( 'DB_NAME', 'wpdb' );
define( 'DB_USER', 'wpuser' );
define( 'DB_PASSWORD', 'wppass' );
define( 'DB_HOST', 'localhost:3306' );
`
	got := parseWpConfig(content)
	want := `{"host":"localhost","dbname":"wpdb","username":"wpuser","password":"wppass","port":"3306"}`
	if got != want {
		t.Fatalf("parseWpConfig\n got: %s\nwant: %s", got, want)
	}
}

func TestParseWpConfigNoPort(t *testing.T) {
	content := `define('DB_NAME','wpdb');define('DB_USER','u');define('DB_PASSWORD','p');define('DB_HOST','localhost');`
	got := parseWpConfig(content)
	want := `{"host":"localhost","dbname":"wpdb","username":"u","password":"p","port":""}`
	if got != want {
		t.Fatalf("parseWpConfig\n got: %s\nwant: %s", got, want)
	}
}

func TestParseWpConfigSocketHost(t *testing.T) {
	// A unix-socket DB_HOST must not be split into host:port.
	content := `define('DB_NAME','wpdb');define('DB_USER','u');define('DB_PASSWORD','p');define('DB_HOST','localhost:/var/run/mysqld/mysqld.sock');`
	got := parseWpConfig(content)
	want := `{"host":"localhost:/var/run/mysqld/mysqld.sock","dbname":"wpdb","username":"u","password":"p","port":""}`
	if got != want {
		t.Fatalf("parseWpConfig socket\n got: %s\nwant: %s", got, want)
	}
}

func TestParseDatabaseURLNoDbName(t *testing.T) {
	// Without a database name the result must be empty so the caller errors out
	// instead of running a dump against no database.
	if got := parseDatabaseURL("DATABASE_URL=mysql://u:p@host:3306/\n"); got != "" {
		t.Fatalf("parseDatabaseURL no dbname = %q, want empty", got)
	}
}

func TestSplitHostPort(t *testing.T) {
	cases := []struct{ in, host, port string }{
		{"db", "db", ""},
		{"db:3306", "db", "3306"},
		{"127.0.0.1:5432", "127.0.0.1", "5432"},
		{"localhost:/tmp/mysql.sock", "localhost:/tmp/mysql.sock", ""},
	}
	for _, c := range cases {
		if h, p := splitHostPort(c.in); h != c.host || p != c.port {
			t.Fatalf("splitHostPort(%q) = (%q,%q), want (%q,%q)", c.in, h, p, c.host, c.port)
		}
	}
}

func TestParsePrestashop(t *testing.T) {
	content := `<?php return array (
  'parameters' => array (
    'database_host' => '127.0.0.1',
    'database_port' => '3306',
    'database_name' => 'psdb',
    'database_user' => 'psuser',
    'database_password' => 'pspass',
  ),
);`
	got := parsePrestashop(content)
	want := `{"host":"127.0.0.1","dbname":"psdb","username":"psuser","password":"pspass","port":"3306"}`
	if got != want {
		t.Fatalf("parsePrestashop\n got: %s\nwant: %s", got, want)
	}
}

func TestParseEmptyReturnsEmpty(t *testing.T) {
	if got := parseMagentoEnv(""); got != "" {
		t.Fatalf("parseMagentoEnv(\"\") = %q, want empty", got)
	}
	if got := parseDatabaseURL("FOO=bar\n"); got != "" {
		t.Fatalf("parseDatabaseURL no url = %q, want empty", got)
	}
	if got := parseWpConfig("<?php // nothing"); got != "" {
		t.Fatalf("parseWpConfig empty = %q, want empty", got)
	}
}
