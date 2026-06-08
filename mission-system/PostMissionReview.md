# Post-Mission Review

After DONE, generate a short post-mission review.

## Questions

1. What worked?
2. What broke?
3. What did we learn?
4. What should be added to doctrine?
5. What should be documented?
6. What should Sector Z remember?
7. Did any agent route incorrectly?
8. Did any agent miss evidence?
9. Did the risk gates work?
10. What should improve next time?

## Output Format

```md
# Post-Mission Review

## What Worked

## What Broke

## Lessons Learned

## Doctrine Updates

## Documentation Updates

## Sector Z Memory

## Agent Performance Notes

## Follow-Up Missions
```

## Rule

Every MEDIUM, LARGE, or CRITICAL mission should produce a post-mission review.

TINY and SMALL missions may skip unless something unexpected happened.

Post-mission reviews feed into:

- Doctrine evolution
- Agent tuning
- Mission Memory
- Future routing improvements
