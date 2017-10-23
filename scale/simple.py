from locust import HttpLocust, TaskSet, task

class SimpleBahavior(TaskSet):
    @task(1)
    def foo(self):
        self.client.get("/foo")

class WebsiteUser(HttpLocust):
    task_set = SimpleBahavior
    min_wait = 2000
    max_wait = 5000