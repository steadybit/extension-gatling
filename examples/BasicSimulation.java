import io.gatling.javaapi.core.*;
import io.gatling.javaapi.http.*;

import static io.gatling.javaapi.core.CoreDsl.*;
import static io.gatling.javaapi.http.HttpDsl.*;

public class BasicSimulation extends Simulation {

	ScenarioBuilder scn = scenario("Basic Example")
		.exec(http("Get README.md")
			.get("https://raw.githubusercontent.com/steadybit/extension-gatling/refs/heads/main/README.md")
			.check(status().is(200))
		);

	{
		setUp(scn.injectOpen(atOnceUsers(1)));
	}
}
