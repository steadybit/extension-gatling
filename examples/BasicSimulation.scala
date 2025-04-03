import io.gatling.core.scenario.Simulation
import io.gatling.core.Predef._
import io.gatling.http.Predef._
import scala.concurrent.duration._

class BasicSimulation extends Simulation {

  val scn = scenario("Basic Example")
    .exec(
      http("Get README.md")
        .get("https://raw.githubusercontent.com/steadybit/extension-gatling/refs/heads/main/README.md")
        .check(status.is(200))
    )

  setUp(
    scn.inject(atOnceUsers(1))
  )
}
