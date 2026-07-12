import { Controller, Get, Post } from "@nestjs/common";

@Controller("users")
export class UserController {
  @Get(":id")
  getUser(id: string) {
    return null;
  }

  @Post()
  createUser() {
    return null;
  }
}
