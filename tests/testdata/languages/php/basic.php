<?php

namespace Example\App;

use Psr\Log\LoggerInterface;

class Service {
    private string $token;

    public function __construct(string $token) {
        $this->token = $token;
    }

    public function run(): void {}
}

function helper(): void {}
