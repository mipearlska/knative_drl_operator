from locust import HttpUser, TaskSet, task, between, constant
from locust import LoadTestShape

class QuickstartUser(HttpUser):
    wait_time = constant(0)

    @task(1)
    def index_page(self):
        self.client.get("/test")

class StagesShape(LoadTestShape):
    """
    A simply load test shape class that has different user and spawn_rate at
    different stages.
    Keyword arguments:
        stages -- A list of dicts, each representing a stage with the following keys:
            duration -- When this many seconds pass the test is advanced to the next stage
            users -- Total user count
            spawn_rate -- Number of users to start/stop per second
            stop -- A boolean that can stop that test at a specific stage
        stop_at_end -- Can be set to stop once all stages have run.
    """

    stages = [
        {"duration": 120, "users": 33, "spawn_rate": 33},
        {"duration": 240, "users": 34, "spawn_rate": 34},
        {"duration": 360, "users": 50, "spawn_rate": 50},
        {"duration": 480, "users": 66, "spawn_rate": 66},
        {"duration": 600, "users": 66, "spawn_rate": 66},
        {"duration": 720, "users": 69, "spawn_rate": 69},
        {"duration": 840, "users": 94, "spawn_rate": 94},
        {"duration": 960, "users": 53, "spawn_rate": 53},
        {"duration": 1080, "users": 56, "spawn_rate": 56},
        {"duration": 1200, "users": 86, "spawn_rate": 86},
        {"duration": 1320, "users": 100, "spawn_rate": 100},
        {"duration": 1440, "users": 56, "spawn_rate": 56},
        {"duration": 1560, "users": 50, "spawn_rate": 50},
        {"duration": 1680, "users": 69, "spawn_rate": 69},
        {"duration": 1800, "users": 33, "spawn_rate": 33},
        {"duration": 1920, "users": 25, "spawn_rate": 25},
        {"duration": 2040, "users": 28, "spawn_rate": 28},
        {"duration": 2160, "users": 55, "spawn_rate": 55},
        {"duration": 2280, "users": 25, "spawn_rate": 25},
        {"duration": 2400, "users": 17, "spawn_rate": 17},
    ]

    def tick(self):
        run_time = self.get_run_time()

        for stage in self.stages:
            if run_time < stage["duration"]:
                tick_data = (stage["users"], stage["spawn_rate"])
                return tick_data

        return None
