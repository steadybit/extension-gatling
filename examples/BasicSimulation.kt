import io.gatling.javaapi.core.CoreDsl.*
import io.gatling.javaapi.core.Simulation
import io.gatling.javaapi.core.ScenarioBuilder
import io.gatling.javaapi.http.HttpDsl.*

class BasicSimulation : Simulation() {

    private val scn: ScenarioBuilder = scenario("Basic Example")
        .exec(
            http("Get README.md")
                .get("https://raw.githubusercontent.com/steadybit/extension-gatling/refs/heads/main/README.md")
                .check(status().`is`(200))
        )

    init {
        setUp(scn.injectOpen(atOnceUsers(1)))
    }
}
