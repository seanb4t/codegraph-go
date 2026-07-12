using Microsoft.AspNetCore.Mvc;

namespace Example {
    [ApiController]
    [Route("api/[controller]")]
    public class UserController : ControllerBase {
        [HttpGet("{id}")]
        public User GetUser(string id) {
            return null;
        }

        [HttpPost]
        public User CreateUser() {
            return null;
        }
    }
}
